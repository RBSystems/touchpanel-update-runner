package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func loadProject(touchpanel TouchpanelStatus) {
	fmt.Printf("%s Loading Project \n", touchpanel.Address)

	time.Sleep(60 * time.Second) // for some reason we keep getting issues with this. It won't load the project for a while

	fmt.Printf("%s Sending project load\n", touchpanel.Address)
	command := "projectload"
	resp, err := SendTelnetCommand(touchpanel, command, true)
	if err != nil {
		ReportError(touchpanel, err)
		return
	}

	fmt.Printf("%s Return Value: %v\n", touchpanel.Address, resp)

	StartWait(touchpanel)
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

func moveProject(touchpanel TouchpanelStatus) {
	fmt.Printf("%s Moving Project\n", touchpanel.Address)

	filename := filepath.Base(touchpanel.Information.ProjectLocation)
	command := "MOVEFILE /ROMDISK/user/system/" + filename + " /ROMDISK/user/Display"

	resp, err := SendTelnetCommand(touchpanel, command, true)
	if err != nil {
		ReportError(touchpanel, err)
		return
	}

	fmt.Printf("%s Move Return Value: %v\n", touchpanel.Address, resp)

	// Send Reboot command
	command = "reboot"
	resp, err = SendTelnetCommand(touchpanel, command, true)
	if err != nil {
		ReportError(touchpanel, err)
		return
	}

	fmt.Printf("%s Reboot Return Value: %v\n", touchpanel.Address, resp)

	StartWait(touchpanel)
}

// Involved in the validation endpoints
func validateTP(touchpanel TouchpanelStatus) {
	m, err := doValidation(touchpanel, false)
	if err == nil {
		reportSuccess(touchpanel)
		return
	}

	// if the error was just IPTable try it again
	if m["iptable"] == false && m["firmware"] == true && m["project"] == true {
		fmt.Printf("%s iptable not loaded\n", touchpanel.Address)

		if touchpanel.Steps[10].Attempts < 2 {
			touchpanel.Steps[10].Attempts++
			touchpanel.Steps[9].Completed = false

			StartWait(touchpanel)
			return
		}
	}

	errStr := "Validation failed: "
	for k, v := range m {
		if v == false {
			errStr = errStr + ": " + k + " "
		}
	}

	ReportError(touchpanel, errors.New(errStr))
}

func getProjectVersion(touchpanel TouchpanelStatus, retry int) (modelInformation, error) {
	fmt.Printf("%s Getting project info...\n", touchpanel.Address)
	info := modelInformation{}

	rawData, err := SendTelnetCommand(touchpanel, "xget ~.LocalInfo.vtpage", true)
	if err != nil {
		return info, err
	}

	// We've tried to retrieve the vtpage at the same time as someone else. Wait for
	// them to finish and try again
	if strings.Contains(rawData, ":Could not") && retry < 2 {
		fmt.Printf("%s Could not get project information, trying again in 45 seconds...\n", touchpanel.Address)
		time.Sleep(45 * time.Second)
		return getProjectVersion(touchpanel, retry+1)
	}

	re := regexp.MustCompile("VTZ=(.*?)\\nDate=(.*?)\\n") // we just want project title and date

	matches := re.FindStringSubmatch(string(rawData))
	if matches == nil {
		fmt.Printf("%s %s\n", touchpanel.Address, rawData)
		return info, errors.New("Bad data returned")
	}

	fmt.Printf("%s Project Info: %+v\n", touchpanel.Address, matches)

	info.ProjectLocation = strings.TrimSpace(matches[1])
	info.ProjectDate = strings.TrimSpace(matches[2])

	fmt.Printf("%s ProjectDate:%s  ProjectName: %s\n", touchpanel.Address, info.ProjectDate, info.ProjectLocation)

	return info, nil
}

func getFirmwareVersion(touchpanel TouchpanelStatus) (string, error) {
	fmt.Printf("%s Getting Firmware Version\n", touchpanel.Address)

	data, err := SendTelnetCommand(touchpanel, "ver", true)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile("\\[v(.*?)\\s")

	match := re.FindStringSubmatch(string(data))
	if match == nil {
		return "", errors.New("Bad data returned")
	}

	fmt.Printf("%s Firmware Version: %s\n", touchpanel.Address, match[1])

	return match[1], nil
}

func InitializeTouchpanel(touchpanel TouchpanelStatus) error {
	fmt.Printf("%s Intializing\n", touchpanel.Address)
	req := TelnetRequest{Address: touchpanel.Address, Command: "initialize", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS")+"/confirmed", "application/json", bytes.NewBuffer(bits))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	// Wait for the touchpanel to come back from initialize
	err = StartWait(touchpanel)
	if err != nil {
		return err
	}

	return nil
}

// ValidateNeedForUpdate checks to make sure the device in question is a TecHD touch panel and needs and update
// Bypassed in final validation after all other steps have suceeded (firmware installed, IP tables, etc.)
func ValidateNeedForUpdate(touchpanel TouchpanelStatus, ignoreTP bool) (bool, string) {
	prompt, err := GetPrompt(touchpanel)
	if err != nil {
		return false, "Couldn't get a prompt"
	}

	fmt.Printf("%s Prompt returned was: %s \n", touchpanel.Address, prompt)

	if touchpanel.Type != "TECHD" || !strings.Contains(prompt, "TSW-750>") {
		return false, "Not a touchpanel. Prompt received: " + prompt
	}

	if touchpanel.Force {
		fmt.Printf("%s Forced update\n", touchpanel.Address)
		return true, ""
	}

	m, err := doValidation(touchpanel, ignoreTP)
	if err != nil {
		fmt.Printf("%s Validation error: %s\n", touchpanel.Address, err.Error())
	} else {
		return false, "Already has firmware and project"
	}

	for k, v := range m {
		if !v {
			fmt.Printf("%s Needs %s\n", touchpanel.Address, k)
		}
	}

	return true, ""
}

// Called from validateTP
func doValidation(touchpanel TouchpanelStatus, ignoreTP bool) (map[string]bool, error) {
	toReturn := make(map[string]bool)
	needed := false
	// We need to validate IPTable, Firmware, and Project
	projVer, err := getProjectVersion(touchpanel, 0)

	if err != nil || !strings.EqualFold(projVer.ProjectDate, touchpanel.Information.ProjectDate) {
		fmt.Printf("%s Return Ver: %s\n", touchpanel.Address, projVer.ProjectDate)
		fmt.Printf("%s Needed Ver: %s\n", touchpanel.Address, touchpanel.ProjectDate)
		if err != nil {
			fmt.Printf("%s ERROR: %s\n", touchpanel.Address, err.Error())
		}

		toReturn["project"] = false
		needed = true
	} else {
		toReturn["project"] = true
	}

	firmware, err := getFirmwareVersion(touchpanel)
	if err != nil || firmware != touchpanel.Information.FirmwareVersion {
		toReturn["firmware"] = false
		needed = true
	} else {
		toReturn["firmware"] = true
	}

	if !ignoreTP {
		ipTable, _ := getIPTable(touchpanel.Address)

		fmt.Printf("%s IPTABLE: %s\n", touchpanel.Address, ipTable)

		if !ipTable.Equals(touchpanel.IPTable) {
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

func CopyProject(touchpanel TouchpanelStatus) {
	fmt.Printf("%s Clearing old project...\n", touchpanel.Address)
	SendTelnetCommand(touchpanel, "delete /ROMDISK/user/Display/*", true) // clear out space for the copy to succeed

	fmt.Printf("%s Submitting to copy Project\n", touchpanel.Address)
	SendFTPRequest(touchpanel, "/FIRMWARE", touchpanel.Information.ProjectLocation)
}

func SendFirmware(touchpanel TouchpanelStatus) {
	fmt.Printf("%s Submitting to move Firmware\n", touchpanel.Address)
	SendFTPRequest(touchpanel, "/FIRMWARE", touchpanel.Information.FirmwareLocation)
}

func RemoveOldPUF(ipAddress string) error {
	req := TelnetRequest{Address: ipAddress, Command: "cd /ROMDISK/user/sytem\nerase *.puf", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS")+"/command", "application/json", bytes.NewBuffer(bits))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

func ReloadIPTable(touchpanel TouchpanelStatus) {
	// Verify that we actually need to reload
	table, err := getIPTable(touchpanel.Address)
	if err == nil && touchpanel.IPTable.Equals(table) {
		step, _ := touchpanel.GetCurrentStep()
		touchpanel.Steps[step].Info = "No need to update--IPTable already matches"

		EvaluateNextStep(touchpanel)
	}

	for i := range touchpanel.IPTable.Entries {
		var status []string
		entry := touchpanel.IPTable.Entries[i]

		command := ""

		if entry.Type == "Gway" {
			command = "ADDMaster " + entry.CipID + " " + entry.AddressSitename
		} else {
			command = "ADDSlave " + entry.CipID + " " + entry.AddressSitename
		}

		response, err := SendTelnetCommand(touchpanel, command, true)
		if err != nil {
			status = append(status, "Error: "+err.Error())
		} else {
			status = append(status, response)
		}
	}

	EvaluateNextStep(touchpanel)
}

func RetrieveIPTable(touchpanel TouchpanelStatus) {
	ipTable, err := getIPTable(touchpanel.Address)
	if err != nil {
		fmt.Printf("%s ERROR: %s\n", touchpanel.Address, err.Error())
		ReportError(touchpanel, err)
		return
	}

	touchpanel.IPTable = ipTable
	EvaluateNextStep(touchpanel)
}

func UpdateFirmware(touchpanel TouchpanelStatus) {
	fmt.Printf("%s Firmware Update \n", touchpanel.Address)

	resp, err := SendTelnetCommand(touchpanel, "puf", true)
	if err != nil {
		ReportError(touchpanel, err)
		return
	}

	fmt.Printf("%s Return Value: %v\n", touchpanel.Address, resp)

	StartWait(touchpanel)
}

func RemoveOldFirmware(touchpanel TouchpanelStatus) {
	err := RemoveOldPUF(touchpanel.Address)
	if err != nil {
		fmt.Printf("%s ERROR: %s\n", touchpanel.Address, err.Error())
		ReportError(touchpanel, err)
		return
	}

	EvaluateNextStep(touchpanel)
}
