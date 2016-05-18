package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func Initialize(ipAddress string, tp TouchpanelStatus) error {
	fmt.Printf("%s Intializing\n", ipAddress)
	req := TelnetRequest{IPAddress: ipAddress, Command: "initialize", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS")+"Confirm", "application/json", bytes.NewBuffer(bits))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	// Wait for it to come back from initialize
	err = startWait(tp)
	if err != nil {
		return err
	}

	return nil
}

func CopyProject(tp TouchpanelStatus) {
	fmt.Printf("%s Clearing old project...\n", tp.IPAddress)
	SendCommand(tp, "delete /ROMDISK/user/Display/*", true) // clear out space for the copy to succeed

	fmt.Printf("%s Submitting to copy Project.\n", tp.IPAddress)
	SendFTPRequest(tp, "/FIRMWARE", tp.Information.ProjectLocation)
}

func SendFirmware(tp TouchpanelStatus) {
	fmt.Printf("%s Submitting to move Firmware.\n", tp.IPAddress)
	SendFTPRequest(tp, "/FIRMWARE", tp.Information.FirmwareLocation)
}

func RemoveOldPUF(ipAddress string) error {
	req := TelnetRequest{IPAddress: ipAddress, Command: "cd /ROMDISK/user/sytem\nerase *.puf", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS"), "application/json", bytes.NewBuffer(bits))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

func ReloadIPTable(tp TouchpanelStatus) {
	// Verify that we actually need to reload
	table, err := getIPTable(tp.IPAddress)

	if err == nil && tp.IPTable.Equals(table) {
		step, _ := tp.GetCurrentStep()
		tp.Steps[step].Info = "No need to update. IPTable already matches."

		EvaluateNextStep(tp)
	}

	for i := range tp.IPTable.Entries {
		entry := tp.IPTable.Entries[i]

		var resp string
		var err error
		var status []string

		if entry.Type == "Gway" {
			command := "ADDMaster " + entry.CipID + " " + entry.IPAddressSitename
			resp, err = SendCommand(tp, command, true)
		} else {
			command := "ADDSlave " + entry.CipID + " " + entry.IPAddressSitename
			resp, err = SendCommand(tp, command, true)
		}

		if err != nil {
			status = append(status, "Error: "+err.Error())
		} else {
			status = append(status, resp)
		}
	}

	EvaluateNextStep(tp)
}

func RetrieveIPTable(tp TouchpanelStatus) {
	ipTable, err := getIPTable(tp.IPAddress)

	if err != nil {
		// TODO: Decide what to do here
		fmt.Printf("%s ERROR: %s\n", tp.IPAddress, err.Error())
		reportError(tp, err)
		return
	}
	tp.IPTable = ipTable
	// fmt.Printf("%s Got the IPtable: %s\n", tp.IPAddress, ipTable)
	EvaluateNextStep(tp)
}

func UpdateFirmware(tp TouchpanelStatus) {
	fmt.Printf("%s Firmware Update \n", tp.IPAddress)

	resp, err := SendCommand(tp, "puf", true)
	if err != nil {
		reportError(tp, err)
		return
	}

	fmt.Printf("%s Return Value: %v\n", tp.IPAddress, resp)

	startWait(tp)
}

func RemoveOldFirmware(tp TouchpanelStatus) {
	err := RemoveOldPUF(tp.IPAddress)

	if err != nil {
		// TODO: Decide what to do here
		fmt.Printf("%s ERROR: %s\n", tp.IPAddress, err.Error())
		reportError(tp, err)
		return
	}

	EvaluateNextStep(tp)
}
