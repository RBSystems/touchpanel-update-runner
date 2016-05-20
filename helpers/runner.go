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

// StartRun starts the touchpanel update process
func StartRun(currentTouchpanel TouchpanelStatus) {
	currentTouchpanel.Attempts++

	currentTouchpanel.Steps = GetTouchpanelSteps()
	currentTouchpanel.Attempts = 0 // We haven't tried yet

	// Get the hostname
	response, err := SendTelnetCommand(currentTouchpanel, "hostname", true)
	if err != nil {
		fmt.Printf("Could not retrieve hostname")
	}

	if strings.Contains(response, "Host Name:") {
		response = strings.Split(response, "Host Name:")[1]
	}

	currentTouchpanel.Hostname = strings.TrimSpace(response)

	UpdateChannel <- currentTouchpanel

	EvaluateNextStep(currentTouchpanel)
}

func StartWait(currentTouchpanel TouchpanelStatus) error {
	fmt.Printf("%s Sending to wait\n", currentTouchpanel.Address)

	req := WaitRequest{Address: currentTouchpanel.Address, Port: 41795, CallbackAddress: os.Getenv("TOUCHPANEL_UPDATE_RUNNER_ADDRESS") + "/callbacks/afterWait"}

	req.Identifier = currentTouchpanel.UUID

	bits, _ := json.Marshal(req)

	// We have to wait for the touchpanel to actually restart otherwise we'll return before it can communicate
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

func reportNotNeeded(touchpanel TouchpanelStatus, status string) {
	fmt.Printf("%s Not needed\n", touchpanel.Address)

	touchpanel.CurrentStatus = status
	touchpanel.EndTime = time.Now()
	UpdateChannel <- touchpanel

	SendToElastic(touchpanel, 0)
}

func reportSuccess(touchpanel TouchpanelStatus) {
	fmt.Printf("%s Success!\n", touchpanel.Address)

	touchpanel.CurrentStatus = "Success"
	touchpanel.EndTime = time.Now()
	UpdateChannel <- touchpanel

	SendToElastic(touchpanel, 0)
}

func ReportError(touchpanel TouchpanelStatus, err error) {
	fmt.Printf("%s Reporting a failure: %s\n", touchpanel.Address, err.Error())

	ipTable := false

	// if we want to retry
	fmt.Printf("%s Attempts: %v\n", touchpanel.Address, touchpanel.Attempts)
	if touchpanel.Attempts < 2 {
		touchpanel.Attempts++

		fmt.Printf("%s Retrying process\n", touchpanel.Address)
		if touchpanel.Steps[0].Completed {
			ipTable = true
		}

		touchpanel.Steps = GetTouchpanelSteps() // Reset the steps

		if ipTable { // If the iptable was already populated
			touchpanel.Steps[0].Completed = true
		}

		UpdateChannel <- touchpanel

		StartWait(touchpanel) // There's no way to know what state we're in, run a wait
		return
	}

	touchpanel.CurrentStatus = "Error"
	touchpanel.EndTime = time.Now()
	touchpanel.ErrorInfo = append(touchpanel.ErrorInfo, err.Error())
	UpdateChannel <- touchpanel

	SendToElastic(touchpanel, 0)
}
