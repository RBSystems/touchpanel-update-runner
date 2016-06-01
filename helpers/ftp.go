package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type FTPRequest struct {
	// Required fields
	DestinationAddress   string `json:",omitempty"`
	DestinationDirectory string `json:",omitempty"`
	FileLocation         string `json:",omitempty"`
	Filename             string `json:",omitempty"`
	CallbackAddress      string `json:",omitempty"`

	// Optional Fields
	CallbackIdentifier string `json:",omitempty"`
	Timeout            int    `json:",omitempty"`
	UsernameFTP        string `json:",omitempty"`
	PasswordFTP        string `json:",omitempty"`

	// Fields not expected in request, will be filled by the service
	SubmissionTime time.Time
	CompletionTime time.Time
	Status         string
	Error          string
}

func SendFTPRequest(touchpanel TouchpanelStatus, path string, file string) error {
	request := FTPRequest{
		DestinationAddress:   touchpanel.Address,
		DestinationDirectory: path,
		FileLocation:         file,
		CallbackAddress:      os.Getenv("TOUCHPANEL_UPDATE_RUNNER_ADDRESS") + "/callback/ftp",
		CallbackIdentifier:   touchpanel.UUID,
	}

	requestJSON, err := json.Marshal(&request)
	if err != nil {
		return err
	}

	response, err := http.Post(os.Getenv("FTP_MICROSERVICE_ADDRESS")+"/send", "application/json", bytes.NewBuffer(requestJSON))
	if err != nil {
		ReportError(touchpanel, err)
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	fmt.Printf("%s Submission response: %s\n", touchpanel.Address, body)
	return nil
}
