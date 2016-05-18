package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/byuoitav/telnet-microservice/helpers"
	"github.com/byuoitav/touchpanel-update-runner/packages/crestron"
)

func reloadIPTable(tp TouchpanelStatus) {
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
			resp, err = helpers.SendCommand(tp, command, true)
		} else {
			command := "ADDSlave " + entry.CipID + " " + entry.IPAddressSitename
			resp, err = helpers.SendCommand(tp, command, true)
		}

		if err != nil {
			status = append(status, "Error: "+err.Error())
		} else {
			status = append(status, resp)
		}
	}

	EvaluateNextStep(tp)
}

func loadProject(tp TouchpanelStatus) {
	fmt.Printf("%s Loading Project \n", tp.IPAddress)

	time.Sleep(60 * time.Second) // for some reason we keep getting issues with this. It won't load the project for a while

	fmt.Printf("%s Sending project load.\n", tp.IPAddress)
	command := "projectload"
	resp, err := helpers.SendCommand(tp, command, true)

	if err != nil {
		reportError(tp, err)
		return
	}
	fmt.Printf("%s Return Value: %v\n", tp.IPAddress, resp)

	startWait(tp)
}

// Send the response of a telnet command to validate success, will return true
// if output is consistent with success, false if need to retry
func validateCommand(output string, command string) bool {
	// List of responses that always denote a retry
	var badResponses = []string{
		"Bad or Incomplete Command",
		"Move Failed",
		"i/o timeout",
	}

	for i := range badResponses {
		if strings.Contains(output, badResponses[i]) {
			return false
		}
	}

	// TODO: Do command-specific checking here

	return true
}

func moveProject(tp TouchpanelStatus) {
	fmt.Printf("%s Moving Project\n", tp.IPAddress)

	filename := filepath.Base(tp.Information.ProjectLocation)
	command := "MOVEFILE /ROMDISK/user/system/" + filename + " /ROMDISK/user/Display"

	resp, err := helpers.SendCommand(tp, command, true)

	if err != nil {
		reportError(tp, err)
		return
	}
	fmt.Printf("%s Move Return Value: %v\n", tp.IPAddress, resp)

	// Send Reboot command
	command = "reboot"
	resp, err = helpers.SendCommand(tp, command, true)

	if err != nil {
		reportError(tp, err)
		return
	}

	fmt.Printf("%s Reboot Return Value: %v\n", tp.IPAddress, resp)

	startWait(tp)
}

func copyProject(tp TouchpanelStatus) {
	fmt.Printf("%s Clearing old project...\n", tp.IPAddress)
	helpers.SendCommand(tp, "delete /ROMDISK/user/Display/*", true) // clear out space for the copy to succeed

	fmt.Printf("%s Submitting to copy Project.\n", tp.IPAddress)
	sendFTPRequest(tp, "/FIRMWARE", tp.Information.ProjectLocation)
}

func sendFirmware(tp TouchpanelStatus) {
	fmt.Printf("%s Submitting to move Firmware.\n", tp.IPAddress)
	sendFTPRequest(tp, "/FIRMWARE", tp.Information.FirmwareLocation)
}

func sendFTPRequest(tp TouchpanelStatus, path string, file string) {
	reqInfo := ftpRequest{ // our request
		IPAddressHostname: tp.IPAddress,
		CallbackAddress:   os.Getenv("TOUCHPANEL_UPDATE_RUNNER_ADDRESS") + "/callbacks/afterFTP",
		Path:              path,
		File:              file,
		Identifier:        tp.UUID}

	b, _ := json.Marshal(&reqInfo)

	resp, err := http.Post(os.Getenv("FTP_MICROSERVICE_ADDRESS"), "application/json", bytes.NewBuffer(b))

	if err != nil {
		reportError(tp, err)
	}
	defer resp.Body.Close()
	b, _ = ioutil.ReadAll(resp.Body)

	fmt.Printf("%s Submission response: %s\n", tp.IPAddress, b)
}

func retrieveIPTable(tp TouchpanelStatus) {
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

func updateFirmware(tp TouchpanelStatus) {
	fmt.Printf("%s Firmware Update \n", tp.IPAddress)

	resp, err := helpers.SendCommand(tp, "puf", true)
	if err != nil {
		reportError(tp, err)
		return
	}

	fmt.Printf("%s Return Value: %v\n", tp.IPAddress, resp)

	startWait(tp)
}

func removeOldFirmware(tp TouchpanelStatus) {
	err := removeOldPUF(tp.IPAddress)

	if err != nil {
		// TODO: Decide what to do here
		fmt.Printf("%s ERROR: %s\n", tp.IPAddress, err.Error())
		reportError(tp, err)
		return
	}

	EvaluateNextStep(tp)
}

func getPrompt(tp TouchpanelStatus) (string, error) {
	var req = TelnetRequest{IPAddress: tp.IPAddress, Command: "hostname"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS")+"/getPrompt", "application/json", bytes.NewBuffer(bits))
	if err != nil {
		return "", err
	}

	b, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}

	respValue := TelnetRequest{}

	err = json.Unmarshal(b, &respValue)

	return respValue.Prompt, nil
}

func initializeTP(tp TouchpanelStatus) {
	err := crestron.Initialize(tp.IPAddress)

	if err != nil {
		// TODO: Decide what to do here
		fmt.Printf("%s ERROR: %s\n", tp.IPAddress, err.Error())
		reportError(tp, err)
		return
	}

	// wait for it to come back from initialize
	err = startWait(tp)
	if err != nil {
		// TODO: Decide what to do here
		fmt.Printf("%s ERROR: %s\n", tp.IPAddress, err.Error())
		reportError(tp, err)
		return
	}
}

// Involved in the validation endpoints
func validateTP(tp TouchpanelStatus) {
	m, err := doValidation(tp, false)
	if err == nil {
		reportSuccess(tp)
		return
	}

	// if the error was just IPTable try it again
	if m["iptable"] == false && m["firmware"] == true && m["project"] == true {
		fmt.Printf("%s iptable not loaded\n", tp.IPAddress)

		if tp.Steps[10].Attempts < 2 {
			tp.Steps[10].Attempts++
			tp.Steps[9].Completed = false

			startWait(tp)
			return
		}
	}

	errStr := "Validation failed: "
	for k, v := range m {
		if v == false {
			errStr = errStr + ": " + k + " "
		}
	}

	reportError(tp, errors.New(errStr))
}

// Called from validateTP
func doValidation(tp TouchpanelStatus, ignoreTP bool) (map[string]bool, error) {
	toReturn := make(map[string]bool)
	needed := false
	// we need to validate IPTable, Firmware, and Project
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
	if !ignoreTP {
		ipTable, _ := getIPTable(tp.IPAddress)

		fmt.Printf("%s IPTABLE: %s\n", tp.IPAddress, ipTable)

		if !ipTable.Equals(tp.IPTable) {
			toReturn["iptable"] = false
			needed = true
		} else {
			toReturn["iptable"] = true
		}
	}
	if needed {
		return toReturn, errors.New("Needed update")
	}

	return toReturn, nil
}

func getProjectVersion(tp TouchpanelStatus, retry int) (modelInformation, error) {
	fmt.Printf("%s Getting project info...\n", tp.IPAddress)
	info := modelInformation{}

	rawData, err := helpers.SendCommand(tp, "xget ~.LocalInfo.vtpage", true)

	if err != nil {
		return info, err
	}

	// We've tried to retrieve the vtpage at the same time as someone else. Wait for
	// them to finish and try again
	if strings.Contains(rawData, ":Could not") && retry < 2 {
		fmt.Printf("%s Could not get project information, trying again in 45 seconds...\n", tp.IPAddress)
		time.Sleep(45 * time.Second)
		return getProjectVersion(tp, retry+1)
	}

	re := regexp.MustCompile("VTZ=(.*?)\\nDate=(.*?)\\n") // we just want project title and date

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

func getFirmwareVersion(tp TouchpanelStatus) (string, error) {
	fmt.Printf("%s Getting Firmware Version\n", tp.IPAddress)

	data, err := helpers.SendCommand(tp, "ver", true)

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

// Checks to make sure the device in question is a TecHD touch panel
// Bypassed in final validation after all other steps have suceeded (firmware installed, IP tables, etc.)
func validateNeed(tp TouchpanelStatus, ignoreTP bool) (bool, string) {
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

	m, err := doValidation(tp, ignoreTP)

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
