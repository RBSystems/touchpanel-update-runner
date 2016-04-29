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

// Starts the TP Update process.
func startRun(curTP tpStatus) {
	curTP.Attempts++

	curTP.Steps = getTPSteps()

	curTP.Attempts = 0 // We haven't tried yet

	// Get the hostname

	response, err := sendCommand(curTP, "hostname", true)

	if err != nil {
		fmt.Printf("Could not retrieve hostname.")
	}
	if strings.Contains(response, "Host Name:") {
		response = strings.Split(response, "Host Name:")[1]
	}

	curTP.Hostname = strings.TrimSpace(response)

	updateChannel <- curTP

	evaluateNextStep(curTP)
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

	defer resp.Body.Close()

	if !strings.Contains(string(body), "Added to queue") {
		return errors.New(string(body))
	}

	return nil
}

func initialize(ipAddress string, config configuration) error {
	fmt.Printf("%s Intializing\n", ipAddress)
	var req = telnetRequest{IPAddress: ipAddress, Command: "initialize", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(config.TelnetServiceLocation+"Confirm", "application/json", bytes.NewBuffer(bits))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	return nil
}

func removeOldPUF(ipAddress string, config configuration) error {
	var req = telnetRequest{IPAddress: ipAddress, Command: "cd /ROMDISK/user/sytem\nerase *.puf", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(config.TelnetServiceLocation, "application/json", bytes.NewBuffer(bits))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

func reportNotNeeded(tp tpStatus, status string) {
	fmt.Printf("%s Not needed\n", tp.IPAddress)

	tp.CurrentStatus = status
	tp.EndTime = time.Now()
	updateChannel <- tp

	sendToELK(tp, 0)
}

func reportSuccess(tp tpStatus) {
	fmt.Printf("%s Success!\n", tp.IPAddress)

	tp.CurrentStatus = "Success"
	tp.EndTime = time.Now()
	updateChannel <- tp

	sendToELK(tp, 0)
}

func reportError(tp tpStatus, err error) {

	fmt.Printf("%s Reporting a failure  %s ...\n", tp.IPAddress, err.Error())

	ipTable := false

	//if we want to retry
	fmt.Printf("%s Attempts: %v, Limit: %v\n", tp.IPAddress, tp.Attempts, config.AttemptLimit)
	if tp.Attempts < config.AttemptLimit {
		tp.Attempts++

		fmt.Printf("%s Retring process.\n", tp.IPAddress)
		if tp.Steps[0].Completed {
			ipTable = true
		}

		tp.Steps = getTPSteps() //reset the steps

		if ipTable { //if the iptable was already populated.
			tp.Steps[0].Completed = true
		}

		updateChannel <- tp

		startWait(tp, config) //Who knows what state, run a wait on them.
		return
	}

	tp.CurrentStatus = "Error"
	tp.EndTime = time.Now()
	tp.ErrorInfo = append(tp.ErrorInfo, err.Error())
	updateChannel <- tp

	sendToELK(tp, 0)
}

func getIPTable(IPAddress string) (IPTable, error) {
	var toReturn = IPTable{}
	//TODO: Make the prompt generic
	var req = telnetRequest{IPAddress: IPAddress, Command: "iptable"}

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

func sendToELK(tp tpStatus, retry int) {
	b, _ := json.Marshal(&tp)

	resp, err := http.Post(config.ESAddress+tp.Batch+"/"+tp.Hostname, "application/json", bytes.NewBuffer(b))

	if err != nil {
		if retry < 2 {
			fmt.Printf("%s error posting to ELK %s. Trying again.\n", tp.IPAddress, err.Error())
			sendToELK(tp, retry+1)
			return
		}
		fmt.Printf("%s Could not report to ELK. %s \n", tp.IPAddress, err.Error())
	} else if resp.StatusCode > 299 || resp.StatusCode < 200 {
		fmt.Printf("%s Status Code: %v\n", tp.IPAddress, resp.StatusCode)
		b, _ := ioutil.ReadAll(resp.Body)
		if retry < 2 {
			fmt.Printf("%s error posting to ELK %s. Trying again.\n", tp.IPAddress, string(b))
			sendToELK(tp, retry+1)
			return
		}
		fmt.Printf("%s Could not report to ELK. %s \n", tp.IPAddress, string(b))
		return
	}
	defer resp.Body.Close()
	fmt.Printf("%s Reported to ELK.\n", tp.IPAddress)
}
