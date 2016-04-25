package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
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
		"Get IPTable",              //0
		"CheckCurrentVersion/Date", //1
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
		completeStep(curTP, stepIndx, "Validating")

		go retrieveIPTable(curTP)
	case 1:
		fmt.Printf("%s We've gotten IP Table.\n", curTP.IPAddress)
		need, str := validateNeed(curTP)
		if !need {
			fmt.Printf("%s Not needed: %s\n", curTP.IPAddress, str)
			curTP.CurrentStatus = "Not needed: " + str
			updateChannel <- curTP
			return
		}
		fmt.Printf("%s Done validating.\n", curTP.IPAddress)
		completeStep(curTP, stepIndx, "Removing old Firmware")
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

		go validateTP(curTP)
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

	fmt.Printf("%s Sending project load.\n", tp.IPAddress)
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
			fmt.Printf("%s bad output: %s \n", tp.IPAddress, str)
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
		"Move Failed",
		"i/o timeout"}

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

	updateChannel <- tp
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

func retrieveIPTable(tp tpStatus) {
	ipTable, err := getIPTable(tp.IPAddress)

	if err != nil {
		//TODO: Decide what to do here
		fmt.Printf("%s ERROR: %s\n", tp.IPAddress, err.Error())
		reportError(tp, err)
		return
	}
	tp.IPTable = ipTable
	//fmt.Printf("%s Got the IPtable: %s\n", tp.IPAddress, ipTable)
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

func getPrompt(tp tpStatus) (string, error) {
	var req = telnetRequest{IPAddress: tp.IPAddress, Command: "hostname"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(config.TelnetServiceLocation+"/getPrompt", "application/json", bytes.NewBuffer(bits))

	if err != nil {
		return "", err
	}

	b, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	respValue := telnetRequest{}

	err = json.Unmarshal(b, &respValue)

	return respValue.Prompt, nil
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
	m, err := doValidation(tp)

	if err == nil {
		tp.CurrentStatus = "Success."
		fmt.Printf("%s Success!\n", tp.IPAddress)
		updateChannel <- tp
		//reportSuccess(tp) //TODO: send success to ELK
		return
	}

	// if the error was just IPTable try it again.
	if m["iptable"] == false && m["firmare"] == true && m["project"] == true {
		fmt.Printf("%s iptable not loaded\n", tp.IPAddress)

		if tp.Steps[10].Attempts < 2 {
			tp.Steps[10].Attempts++
			tp.Steps[9].Completed = false

			startWait(tp, config)
			return
		}
	}

	reportError(tp, err)
}

func doValidation(tp tpStatus) (map[string]bool, error) {
	toReturn := make(map[string]bool)
	needed := false
	//we need to validate IPTable, Firmware, and Project
	projVer, err := getProjectVersion(tp, 0)

	if err != nil || !strings.EqualFold(projVer.ProjectDate, tp.Information.ProjectDate) {
		fmt.Printf("%s Return Ver: %s\n", tp.IPAddress, projVer.ProjectDate)
		fmt.Printf("%s Needed Ver: %s\n", tp.IPAddress, tp.ProjectDate)
		if err != nil {
			fmt.Printf("%s ERROR: %s\n", tp.IPAddress, err.Error())
		}
		toReturn["project"] = false
		needed = true
	} else {
		toReturn["project"] = true
	}

	firmware, err := getFirmwareVersion(tp)
	if err != nil || firmware != tp.Information.FirmwareVersion {
		toReturn["firmware"] = false
		needed = true
	} else {
		toReturn["firmware"] = true
	}

	ipTable, _ := getIPTable(tp.IPAddress)

	fmt.Printf("%s IPTABLE: %s\n", tp.IPAddress, ipTable)

	if !ipTable.Equals(tp.IPTable) {
		toReturn["iptable"] = false
		needed = true
	} else {
		toReturn["iptable"] = true
	}

	if needed {
		return toReturn, errors.New("Needed update")
	}

	return toReturn, nil

}

func getProjectVersion(tp tpStatus, retry int) (modelInformation, error) {
	fmt.Printf("%s Getting project info...\n", tp.IPAddress)
	info := modelInformation{}

	rawData, err := sendCommand(tp, "xget ~.LocalInfo.vtpage", true)

	if err != nil {
		return info, err
	}

	//We've tried to retrieve the vtpage at the same time as someone else. Wait for
	//them to finish and try again.
	if strings.Contains(rawData, ":Could not") && retry < 2 {
		fmt.Printf("%s Could not get project information, trying again in 45 seconds...\n", tp.IPAddress)
		time.Sleep(45 * time.Second)
		return getProjectVersion(tp, retry+1)
	}

	re := regexp.MustCompile("VTZ=(.*?)\\nDate=(.*?)\\n") //we just want project title and date

	matches := re.FindStringSubmatch(string(rawData))

	if matches == nil {
		fmt.Printf("%s %s\n", tp.IPAddress, rawData)
		return info, errors.New("Bad data returned.")
	}

	fmt.Printf("%s Project Info: %+v\n", tp.IPAddress, matches)

	info.ProjectLocation = strings.TrimSpace(matches[1])
	info.ProjectDate = strings.TrimSpace(matches[2])

	fmt.Printf("%s ProjectDate:%s  ProjectName: %s\n", tp.IPAddress, info.ProjectDate, info.ProjectLocation)

	return info, nil
}

func getFirmwareVersion(tp tpStatus) (string, error) {
	fmt.Printf("%s Getting Firmware Version\n", tp.IPAddress)

	data, err := sendCommand(tp, "ver", true)

	if err != nil {
		return "", err
	}

	re := regexp.MustCompile("\\[v(.*?)\\s")

	match := re.FindStringSubmatch(string(data))

	if match == nil {
		return "", errors.New("Bad data returned.")
	}

	fmt.Printf("%s Firmware Version: %s\n", tp.IPAddress, match[1])

	return match[1], nil
}

func validateNeed(tp tpStatus) (bool, string) {
	prompt, err := getPrompt(tp)

	fmt.Printf("%s Prompt Returned was: %s \n", tp.IPAddress, prompt)

	if err != nil {
		return false, "Couldn't get a prompt."
	}

	if tp.Type != "TECHD" || !strings.Contains(prompt, "TSW-750>") {
		return false, "Not a touchpanel. Prompt received: " + prompt
	}

	if tp.Force {
		fmt.Printf("%s Forced update.\n", tp.IPAddress)
		return true, ""
	}

	m, err := doValidation(tp)

	if err != nil {
		fmt.Printf("%s Validation error: %s\n", tp.IPAddress, err.Error())
	}

	if err == nil {
		return false, "Already has firmware and project."
	}

	for k, v := range m {
		if !v {
			fmt.Printf("%s needs %s\n", tp.IPAddress, k)
		}
	}

	return true, ""
}
