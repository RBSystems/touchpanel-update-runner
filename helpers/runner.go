package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

// Starts the touchpanel update process
func StartRun(curTP TouchpanelStatus) {
	curTP.Attempts++

	curTP.Steps = GetTouchpanelSteps()
	curTP.Attempts = 0 // We haven't tried yet

	// Get the hostname
	response, err := SendCommand(curTP, "hostname", true)
	if err != nil {
		fmt.Printf("Could not retrieve hostname")
	}

	if strings.Contains(response, "Host Name:") {
		response = strings.Split(response, "Host Name:")[1]
	}

	curTP.Hostname = strings.TrimSpace(response)

	UpdateChannel <- curTP

	EvaluateNextStep(curTP)
}

func startWait(curTP TouchpanelStatus) error {
	fmt.Printf("%s Sending to wait\n", curTP.IPAddress)

	var req = WaitRequest{IPAddressHostname: curTP.IPAddress, Port: 41795, CallbackAddress: os.Getenv("TOUCHPANEL_UPDATE_RUNNER_ADDRESS") + "/callbacks/afterWait"}

	req.Identifier = curTP.UUID

	bits, _ := json.Marshal(req)

	// we have to wait for the thing to actually restart - otherwise we'll return before it gets in a non-communicative state
	time.Sleep(10 * time.Second) // TODO: Shift this into our wait microservice

	resp, err := http.Post(os.Getenv("WAIT_FOR_REBOOT_MICROSERVICE_ADDRESS"), "application/json", bytes.NewBuffer(bits))

	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)

	defer resp.Body.Close()

	if !strings.Contains(string(body), "Added to queue") {
		return errors.New(string(body))
	}

	return nil
}

func reportNotNeeded(tp TouchpanelStatus, status string) {
	fmt.Printf("%s Not needed\n", tp.IPAddress)

	tp.CurrentStatus = status
	tp.EndTime = time.Now()
	UpdateChannel <- tp

	SendToElastic(tp, 0)
}

func reportSuccess(tp TouchpanelStatus) {
	fmt.Printf("%s Success!\n", tp.IPAddress)

	tp.CurrentStatus = "Success"
	tp.EndTime = time.Now()
	UpdateChannel <- tp

	SendToElastic(tp, 0)
}

func ReportError(tp TouchpanelStatus, err error) {
	fmt.Printf("%s Reporting a failure  %s ...\n", tp.IPAddress, err.Error())

	ipTable := false

	// if we want to retry
	fmt.Printf("%s Attempts: %v\n", tp.IPAddress, tp.Attempts)
	if tp.Attempts < 2 {
		tp.Attempts++

		fmt.Printf("%s Retring process.\n", tp.IPAddress)
		if tp.Steps[0].Completed {
			ipTable = true
		}

		tp.Steps = GetTouchpanelSteps() // Reset the steps

		if ipTable { // If the iptable was already populated
			tp.Steps[0].Completed = true
		}

		UpdateChannel <- tp

		startWait(tp) // Who knows what state, run a wait on them
		return
	}

	tp.CurrentStatus = "Error"
	tp.EndTime = time.Now()
	tp.ErrorInfo = append(tp.ErrorInfo, err.Error())
	UpdateChannel <- tp

	SendToElastic(tp, 0)
}

func getIPTable(IPAddress string) (IPTable, error) {
	var toReturn = IPTable{}
	// TODO: Make the prompt generic
	var req = TelnetRequest{IPAddress: IPAddress, Command: "iptable"}

	bits, _ := json.Marshal(req)

	resp, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS"), "application/json", bytes.NewBuffer(bits))

	if err != nil {
		return toReturn, err
	}

	bits, err = ioutil.ReadAll(resp.Body)

	if err != nil {
		return toReturn, err
	}

	err = json.Unmarshal(bits, &toReturn)
	if err != nil {
		return toReturn, err
	}

	if len(toReturn.Entries) < 1 {
		return toReturn, errors.New("There were no entries in the IP Table, error.")
	}

	return toReturn, nil
}
