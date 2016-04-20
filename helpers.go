package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
		"Wait",                     //6
		"Wait for Initialization", //7
		"Copy Project",            //8
		"Move Project",            //9
		"Load Project",            //10
		"Wait",                    //11
		"Reload IPTable",          //12
		"Validate")                //13
	return n
}

func evaluateNextStep(curTP tpStatus) {

	//-----TESTING---
	for i := 0; i < 5; i++ {
		curTP.Steps[i].Completed = true
	}
	//---------------

	stepIndx, err := curTP.GetCurStep()

	if err != nil {
		return
	}

	switch stepIndx { //determine where to go next.
	case 3: //Initialize - next is copy firmware
		fmt.Printf("Moving %s to copy firmware.\n", curTP.UUID)

		//Set status and update the
		curTP.Steps[stepIndx].Completed = true
		curTP.CurStatus = "Sending Firmware."
		tpStatusMap[curTP.UUID] = curTP
		fmt.Printf("Step 4 %+v\n", tpStatusMap[curTP.UUID].Steps[stepIndx])

		go sendFirmware(curTP) //ship this off concurrently - don't block.

	case 4:
		fmt.Printf("Moving %s to update firmware.\n", curTP.UUID)

		curTP.Steps[stepIndx].Completed = true
		curTP.CurStatus = "Updating Firmware."
		tpStatusMap[curTP.UUID] = curTP
		fmt.Printf("Step 5 %+v\n", tpStatusMap[curTP.UUID].Steps[stepIndx])

		go updateFirmware(curTP)
	case 5:
		fmt.Printf("Done updating firmware.\n")
		curTP.Steps[stepIndx].Completed = true
		curTP.Steps[stepIndx+1].Completed = true //For now - we need to verify that this step is needed.
		curTP.CurStatus = "Copy Project."
		tpStatusMap[curTP.UUID] = curTP

		go copyProject(curTP)
	case 7:
		fmt.Printf("Project copied\n")
	default:
	}
}

func copyProject(tp tpStatus) {
	fmt.Printf("Submitting to copy Project.\n")
	sendFTPRequest(tp, "/FTP", tp.Information.ProjectLocation)
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

	var req = telnetRequest{IPAddress: tp.IPAddress, Command: "puf", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(config.TelnetServiceLocation, "application/json", bytes.NewBuffer(bits))

	if err != nil {
		reportError(tp, err)
		return
	}

	b, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		reportError(tp, err)
		return
	}

	fmt.Printf("Return Value: %s", b)

	startWait(tp, config)
}
