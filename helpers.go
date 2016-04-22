package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
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
		"Initialzie",               //3
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
	for i := 0; i < 7; i++ {
			curTP.Steps[i].Completed = true
	}
//-----------------------------------------
//                 /DEBUG
//-----------------------------------------

	stepIndx, err := curTP.GetCurStep()

	if err != nil {
		return
	}



	switch stepIndx { //determine where to go next.
	case 3: //Initialize - next is copy firmware
		fmt.Printf("Moving %s to copy firmware.\n", curTP.UUID)

		//Set status and update the
		completeStep(curTP, stepIndx, "Sending Firmware")
		go sendFirmware(curTP) //ship this off concurrently - don't block.

	case 4:
		fmt.Printf("Moving %s to update firmware.\n", curTP.UUID)

		completeStep(curTP, stepIndx, "Updating Firmware")
		go updateFirmware(curTP)
	case 5:
		fmt.Printf("Done updating firmware.\n")
		completeStep(curTP, stepIndx, "Sending Project")

		go copyProject(curTP)
	case 6:
		fmt.Printf("Project copied\n")
		completeStep(curTP, stepIndx, "Moving Project")

		go moveProject(curTP)
	case 7:
		fmt.Printf("Project Moved\n")
		completeStep(curTP, stepIndx, "Loading Project")

		go loadProject(curTP)
	case 8:
		fmt.Printf("Project Loaded\n")
		completeStep(curTP, stepIndx, "Reload IPTable")

		go reloadIPTable(curTP)
	case 9:
		fmt.Printf("IPTable loaded\n")
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
			resp, err = sendCommand(tp, command)
		} else {
			command := "ADDSlave " + entry.CipID + " " + entry.IPAddressSitename
			resp, err = sendCommand(tp, command)
		}

		if err != nil {
			status = append(status, "Error: "+err.Error())
		} else {
			status = append(status, resp)
		}
	}
}

func loadProject(tp tpStatus) {
	fmt.Printf("Loading Project %s \n", tp.IPAddress)

	command := "projectload"
	resp, err := sendCommand(tp, command)

	if err != nil {
		reportError(tp, err)
		return
	}
	fmt.Printf("Return Value: %v\n", resp)

	startWait(tp, config)
}

func sendCommand(tp tpStatus, command string) (string, error) {
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

	step, _ := tp.GetCurStep()
	tp.Steps[step].Info = string(b)

	return string(b), nil
}

func moveProject(tp tpStatus) {
	fmt.Printf("Moving Project %s \n", tp.IPAddress)

	filename := filepath.Base(tp.Information.ProjectLocation)
	command := "MOVEFILE /ROMDISK/user/system/" + filename + " /ROMDISK/user/Display"

	resp, err := sendCommand(tp, command)

	if err != nil {
		reportError(tp, err)
		return
	}
	fmt.Printf("Return Value: %v\n", resp)

	//Send Reboot command
	command = "reboot"
	resp, err = sendCommand(tp, command)

	if err != nil {
		reportError(tp, err)
		return
	}
	fmt.Printf("Return Value: %v\n", resp)

	startWait(tp, config)
}

func completeStep(tp tpStatus, step int, curStatus string) {
	tp.Steps[step].Completed = true
	tp.CurStatus = curStatus
	tpStatusMap[tp.UUID] = tp
}

func copyProject(tp tpStatus) {
	fmt.Printf("Submitting to copy Project.\n")
	sendFTPRequest(tp, "/FIRMWARE", tp.Information.ProjectLocation)
}

func sendFirmware(tp tpStatus) {
	fmt.Printf("Submitting to move Firmware.\n")
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

	fmt.Printf("Request: %s\n", b)

	resp, err := http.Post(config.FTPServiceLocation, "application/json", bytes.NewBuffer(b))

	if err != nil {
		reportError(tp, err)
	}
	b, _ = ioutil.ReadAll(resp.Body)
	fmt.Printf("Submission response: %s\n", b)

}

func updateFirmware(tp tpStatus) {
	fmt.Printf("Firmware Update %s \n", tp.IPAddress)

	resp, err := sendCommand(tp, "puf")

	if err != nil {
		reportError(tp, err)
		return
	}

	fmt.Printf("Return Value: %v\n", resp)

	startWait(tp, config)
}
