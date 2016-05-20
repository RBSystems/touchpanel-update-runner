package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type TelnetRequest struct {
	Address string
	Port    string
	Command string
	Prompt  string
}

func SendTelnetCommand(touchpanel TouchpanelStatus, command string, tryAgain bool) (string, error) { // Sends telnet commands
	var req = TelnetRequest{Address: touchpanel.Address, Command: command, Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS"), "application/json", bytes.NewBuffer(bits))
	if err != nil {
		return "", err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	str := string(b)

	// TODO: Allow for multiple retries
	if !validateCommand(str, command) {
		if tryAgain {
			fmt.Printf("%s bad output: %s \n", touchpanel.Address, str)
			fmt.Printf("%s Retrying command %s ...\n", touchpanel.Address, command)
			str, _ = SendTelnetCommand(touchpanel, command, false) // Try again, but don't report
		} else {
			return "", errors.New("Issue with command: " + str)
		}
	}

	step, _ := touchpanel.GetCurrentStep()
	touchpanel.Steps[step].Info = str

	return str, nil
}

func GetPrompt(touchpanel TouchpanelStatus) (string, error) {
	var req = TelnetRequest{Address: touchpanel.Address, Command: "hostname"}
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
