package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

//Starts the TP Update process.
func startRun(submissionChannel <-chan tpStatus, config configuration) {
	for true {
		curTP := <-submissionChannel
		curTP.Attempts++

		curTP.Steps = getTPSteps()

		tpStatusMap[curTP.UUID] = curTP
		//TODO:Check to validate that the current project version and date
		//validate the need for the update.
		curTP.Steps[0].Completed = true
		//TODO:Validate the IPAddress is a touchpanel

		evaluateNextStep(curTP)
	}
}

func startWait(curTP tpStatus, config configuration) error {
	fmt.Printf("%s Sending to wait\n", curTP.IPAddress)

	var req = waitRequest{IPAddressHostname: curTP.IPAddress, Port: 41795, CallbackAddress: config.Hostname + "/callbacks/afterWait"}

	req.Identifier = curTP.UUID

	bits, _ := json.Marshal(req)

	//fmt.Printf("Payload being send: \n %s \n", string(bits))

	//we have to wait for the thing to actually restart - otherwise we'll return
	//before it gets in a non-communicative state.
	time.Sleep(10 * time.Second) //TODO: Shift this into our wait microservice.

	resp, err := http.Post(config.PauseServiceLocaiton, "application/json", bytes.NewBuffer(bits))

	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)

	if !strings.Contains(string(body), "Added to queue") {
		return errors.New(string(body))
	}

	return nil
}

func initialize(ipAddress string, config configuration) error {
	fmt.Printf("%s Intializing\n", ipAddress)
	var req = telnetRequest{IPAddress: ipAddress, Command: "initialize", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	_, err := http.Post(config.TelnetServiceLocation+"Confirm", "application/json", bytes.NewBuffer(bits))

	if err != nil {
		return err
	}

	return nil
}

func removeOldPUF(ipAddress string, config configuration) error {
	var req = telnetRequest{IPAddress: ipAddress, Command: "cd /ROMDISK/user/sytem\nerase *.puf", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	_, err := http.Post(config.TelnetServiceLocation, "application/json", bytes.NewBuffer(bits))

	if err != nil {
		return err
	}

	return nil
}

func reportError(tp tpStatus, err error) {

	fmt.Printf("%s Reporting a failure...", tp.IPAddress)

	ipTable := false

	//if we want to retry
	if tp.Attempts < config.AttemptLimit {
		tp.Attempts++

		fmt.Printf("%s Retring process.", tp.IPAddress)
		if tp.Steps[1].Completed {
			ipTable = true
		}

		tp.Steps = getTPSteps() //reset the steps

		if ipTable { //if the iptable was already populated.
			tp.Steps[1].Completed = true
		}

		evaluateNextStep(tp)
		return
	}

	tp.CurrentStatus = "Error"
	tp.EndTime = time.Now()
	tp.errorInfo = append(tp.errorInfo, err.Error())
}

func getIPTable(IPAddress string) (IPTable, error) {
	var toReturn = IPTable{}
	//TODO: Make the prompt generic
	var req = telnetRequest{IPAddress: IPAddress, Command: "iptable", Prompt: "TSW-750>"}

	bits, _ := json.Marshal(req)

	resp, err := http.Post(config.TelnetServiceLocation, "application/json", bytes.NewBuffer(bits))

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
