package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

func getTPSteps() []step {
	var steps []step

	names := getTPStepNames()

	for indx := range names {
		steps = append(steps, step{StepName: names[indx], Completed: false})
	}

	return steps
}

//Currently if this changes we need to edit the status updaters and post wait

//TODO: Find a way to make this dynamic (maybe a linked list type thing? Maybe a map
//of steps to their next steps)
func getTPStepNames() []string {
	var n []string
	n = append(n,
		"CheckCurrentVersion/Date", //0
		"Get IPTable",              //1
		"Remove Old Firmware",      //2
		"Initialize",               //3
		"Copy Firmware",            //4
		"Update Firmware",          //5
		"Copy Project",             //6
		"Move Project",             //7
		"Load Project",             //8
		"Reload IPTable",           //9
		"Validate")                 //10
	return n
}

//TODO: Move all steps (0-3) to this paradigm
func evaluateNextStep(curTP tpStatus) {

	//-----------------------------------------
	//                 DEBUG
	//-----------------------------------------
	// for i := 0; i < 7; i++ {
	//			curTP.Steps[i].Completed = true
	//	}
	//-----------------------------------------
	//                 /DEBUG
	//-----------------------------------------

	stepIndx, err := curTP.GetCurStep()

	if err != nil {
		return
	}

	switch stepIndx { //determine where to go next.
	case 0:
		fmt.Printf("%s Done validating.\n", curTP.IPAddress)
		completeStep(curTP, stepIndx, "Getting IP Table")

		go setIPTable(curTP)
	case 1:
		fmt.Printf("%s We've gotten IP Table.\n", curTP.IPAddress)
		completeStep(curTP, stepIndx, "Getting IP Table")

		go removeOldFirmware(curTP)
	case 2:
		fmt.Printf("%s Old Firmware removed.\n", curTP.IPAddress)
		completeStep(curTP, stepIndx, "Initializing")

		go initializeTP(curTP)
	case 3: //Initialize - next is copy firmware
		fmt.Printf("%s Moving to copy firmware.\n", curTP.IPAddress)

		//Set status and update the
		completeStep(curTP, stepIndx, "Sending Firmware")
		go sendFirmware(curTP) //ship this off concurrently - don't block.

	case 4:
		fmt.Printf("%s Moving to update firmware.\n", curTP.IPAddress)

		completeStep(curTP, stepIndx, "Updating Firmware")
		go updateFirmware(curTP)
	case 5:
		fmt.Printf("%s Done updating firmware.\n", curTP.IPAddress)
		completeStep(curTP, stepIndx, "Sending Project")

		go copyProject(curTP)
	case 6:
		fmt.Printf("%s Project copied\n", curTP.IPAddress)
		completeStep(curTP, stepIndx, "Moving Project")

		go moveProject(curTP)
	case 7:
		fmt.Printf("%s Project Moved\n", curTP.IPAddress)
		completeStep(curTP, stepIndx, "Loading Project")

		go loadProject(curTP)
	case 8:
		fmt.Printf("%s Project Loaded\n", curTP.IPAddress)
		completeStep(curTP, stepIndx, "Reload IPTable")

		go reloadIPTable(curTP)
	case 9:
		fmt.Printf("%s IPTable loaded\n", curTP.IPAddress)
		completeStep(curTP, stepIndx, "Validating")

	default:
	}
}

func reloadIPTable(tp tpStatus) {
	//veify that we actually need to reload the thing
	table, err := getIPTable(tp.IPAddress)

	if err == nil && tp.IPTable.Equals(table) {
		//we're done! time to validate
		step, _ := tp.GetCurStep()
		tp.Steps[step].Info = "No need to update. IPTable already matches."

		evaluateNextStep(tp)
	}

	for i := range tp.IPTable.Entries {
		entry := tp.IPTable.Entries[i]

		var resp string
		var err error
		var status []string

		if entry.Type == "Gway" {
			command := "ADDMaster " + entry.CipID + " " + entry.IPAddressSitename
			resp, err = sendCommand(tp, command, true)
		} else {
			command := "ADDSlave " + entry.CipID + " " + entry.IPAddressSitename
			resp, err = sendCommand(tp, command, true)
		}

		if err != nil {
			status = append(status, "Error: "+err.Error())
		} else {
			status = append(status, resp)
		}
	}
	evaluateNextStep(tp)
}

func loadProject(tp tpStatus) {
	fmt.Printf("%s Loading Project \n", tp.IPAddress)

	time.Sleep(60 * time.Second) //for some reason we keep getting issues with this. It won't load the project for a while.

	fmt.Printf("%s Sending project load.", tp.IPAddress)
	command := "projectload"
	resp, err := sendCommand(tp, command, true)

	if err != nil {
		reportError(tp, err)
		return
	}
	fmt.Printf("%s Return Value: %v\n", tp.IPAddress, resp)

	startWait(tp, config)
}

func sendCommand(tp tpStatus, command string, tryAgain bool) (string, error) {
	var req = telnetRequest{IPAddress: tp.IPAddress, Command: command, Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(config.TelnetServiceLocation, "application/json", bytes.NewBuffer(bits))

	if err != nil {
		return "", err
	}

	b, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	str := string(b)

	//TODO: Potentially allow for multiple retries.
	if !validateCommand(str, command) {
		if tryAgain {
			fmt.Printf("%s bad output: %s", tp.IPAddress, str)
			fmt.Printf("%s Retrying command %s ...\n", tp.IPAddress, command)
			str, err = sendCommand(tp, command, false) //Try again, but don't re
		} else {
			return "", errors.New("Issue with command: " + str)
		}
	}

	step, _ := tp.GetCurStep()
	tp.Steps[step].Info = str

	return str, nil
}

//Send the response of a telnet command to validate success, will return true
//if output is consistent with success, false if need to retry.
func validateCommand(output string, command string) bool {

	//List of responses that always denote a retry.
	var generalBad = []string{
		"Bad or Incomplete Command",
		"Move Failed"}

	for i := range generalBad {
		if strings.Contains(output, generalBad[i]) {
			return false
		}
	}
	//Do command specific checking here.

	return true
}

func moveProject(tp tpStatus) {
	fmt.Printf("%s Moving Project\n", tp.IPAddress)

	filename := filepath.Base(tp.Information.ProjectLocation)
	command := "MOVEFILE /ROMDISK/user/system/" + filename + " /ROMDISK/user/Display"

	resp, err := sendCommand(tp, command, true)

	if err != nil {
		reportError(tp, err)
		return
	}
	fmt.Printf("%s Move Return Value: %v\n", tp.IPAddress, resp)

	//Send Reboot command
	command = "reboot"
	resp, err = sendCommand(tp, command, true)

	if err != nil {
		reportError(tp, err)
		return
	}
	fmt.Printf("%s Reboot Return Value: %v\n", tp.IPAddress, resp)

	startWait(tp, config)
}

func completeStep(tp tpStatus, step int, curStatus string) {
	tp.Steps[step].Completed = true
	tp.CurrentStatus = curStatus
	tpStatusMap[tp.UUID] = tp
}

func copyProject(tp tpStatus) {
	fmt.Printf("%s Clearing old project...\n", tp.IPAddress)
	sendCommand(tp, "delete /ROMDISK/user/Display/*", true) //clear out space for the copy to succeed.

	fmt.Printf("%s Submitting to copy Project.\n", tp.IPAddress)
	sendFTPRequest(tp, "/FIRMWARE", tp.Information.ProjectLocation)
}

func sendFirmware(tp tpStatus) {
	fmt.Printf("%s Submitting to move Firmware.\n", tp.IPAddress)
	sendFTPRequest(tp, "/FIRMWARE", tp.Information.FirmwareLocation)
}

func sendFTPRequest(tp tpStatus, path string, file string) {
	reqInfo := ftpRequest{ //our request.
		IPAddressHostname: tp.IPAddress,
		CallbackAddress:   config.Hostname + "/callbacks/afterFTP",
		Path:              path,
		File:              file,
		Identifier:        tp.UUID}

	b, _ := json.Marshal(&reqInfo)

	//fmt.Printf("Request: %s\n", b)

	resp, err := http.Post(config.FTPServiceLocation, "application/json", bytes.NewBuffer(b))

	if err != nil {
		reportError(tp, err)
	}
	b, _ = ioutil.ReadAll(resp.Body)
	fmt.Printf("%s Submission response: %s\n", tp.IPAddress, b)

}

func setIPTable(tp tpStatus) {
	ipTable, err := getIPTable(tp.IPAddress)

	if err != nil {
		//TODO: Decide what to do here
		fmt.Printf("%s\n", err.Error())
		reportError(tp, err)
		return
	}
	fmt.Printf("%s Got the IPtable: %s\n", tp.IPAddress, ipTable)
	evaluateNextStep(tp)
}

func updateFirmware(tp tpStatus) {
	fmt.Printf("%s Firmware Update \n", tp.IPAddress)

	resp, err := sendCommand(tp, "puf", true)

	if err != nil {
		reportError(tp, err)
		return
	}

	fmt.Printf("%s Return Value: %v\n", tp.IPAddress, resp)

	startWait(tp, config)
}

func removeOldFirmware(tp tpStatus) {
	err := removeOldPUF(tp.IPAddress, config)

	if err != nil {
		//TODO: Decide what to do here
		fmt.Printf("%s ERROR: %s\n", tp.IPAddress, err.Error())
		reportError(tp, err)
		return
	}

	evaluateNextStep(tp)
}

func initializeTP(tp tpStatus) {
	err := initialize(tp.IPAddress, config)

	if err != nil {
		//TODO: Decide what to do here
		fmt.Printf("%s ERROR: %s\n", tp.IPAddress, err.Error())
		reportError(tp, err)
		return
	}
	//curTP.CurStatus = "Waiting for post initialize reboot."
	//wait for it to come back from initialize
	err = startWait(tp, config)
	if err != nil {
		//TODO: Decide what to do here
		fmt.Printf("%s ERROR: %s\n", tp.IPAddress, err.Error())
		reportError(tp, err)
		return
	}
}

func validateTP(tp tpStatus) {
	//we need to validate IPTable, Firmware, and Project
	ipTable, _ := getIPTable(tp.IPAddress)

	if ipTable.Equals(tp.IPTable) {

	}

}
