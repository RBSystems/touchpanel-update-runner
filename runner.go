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
		tpStatusMap[curTP.UUID] = curTP
		//TODO:Check to validate that the current project version and date
		//validate the need for the update.
		curTP.Steps[0].Completed = true
		//TODO:Validate the IPAddress is a touchpanel

		ipTable, err := getIPTable(curTP.IPAddress)

		if err != nil {
			//TODO: Decide what to do here
			fmt.Printf("%s\n", err.Error())
			reportError(curTP, err)
			continue
		}
		fmt.Printf("Got the IPtable for %s: %s\n", curTP.IPAddress, ipTable)
		curTP.IPTable = ipTable

		curTP.Steps[1].Completed = true
		curTP.CurStatus = "Remove Old Firmware."
		tpStatusMap[curTP.UUID] = curTP

		//Remove old PUF files
		err = removeOldPUF(curTP.IPAddress, config)
		if err != nil {
			//TODO: Decide what to do here
			fmt.Printf("%s\n", err.Error())
			reportError(curTP, err)
			continue
		}
		curTP.Steps[2].Completed = true
		curTP.CurStatus = "Initializing."
		tpStatusMap[curTP.UUID] = curTP

		//run an initialize
		//err = initialize(curTP.IPAddress, config)
		if err != nil {
			//TODO: Decide what to do here
			fmt.Printf("%s\n", err.Error())
			reportError(curTP, err)
			continue
		}
		//curTP.CurStatus = "Waiting for post initialize reboot."
		//wait for it to come back from initialize
		err = startWait(curTP, config)
		if err != nil {
			//TODO: Decide what to do here
			fmt.Printf("%s\n", err.Error())
			reportError(curTP, err)
			continue
		}

	}
}

func startWait(curTP tpStatus, config configuration) error {
	fmt.Printf("Sending to wait: %s\n", curTP.IPAddress)

	var req = waitRequest{IPAddressHostname: curTP.IPAddress, Port: 41795, CallbackAddress: config.Hostname + "/callbacks/afterWait"}

	req.Identifier = curTP.UUID

	bits, _ := json.Marshal(req)

	//fmt.Printf("Payload being send: \n %s \n", string(bits))
	time.Sleep(5 * time.Second) //TODO: Shift this into our wait microservice.

	resp, err := http.Post(config.PauseServiceLocaiton, "application/json", bytes.NewBuffer(bits))

	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)

	if strings.EqualFold(string(body), "Added to queue.") {
		return errors.New(string(body))
	}

	return nil
}

func initialize(ipAddress string, config configuration) error {
	fmt.Printf("Intializing %s \n", ipAddress)
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
	tp.CurStatus = "Error"
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
