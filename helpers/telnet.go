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

type telnetRequest struct {
	IPAddress string
	Port      string
	Command   string
	Prompt    string
}

func SendCommand(tp tpStatus, command string, tryAgain bool) (string, error) { // Sends telnet commands
	var req = telnetRequest{IPAddress: tp.IPAddress, Command: command, Prompt: "TSW-750>"}
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

	// TODO: Potentially allow for multiple retries
	if !validateCommand(str, command) {
		if tryAgain {
			fmt.Printf("%s bad output: %s \n", tp.IPAddress, str)
			fmt.Printf("%s Retrying command %s ...\n", tp.IPAddress, command)
			str, err = sendCommand(tp, command, false) // Try again, but don't report
		} else {
			return "", errors.New("Issue with command: " + str)
		}
	}

	step, _ := tp.GetCurStep()
	tp.Steps[step].Info = str

	return str, nil
}
