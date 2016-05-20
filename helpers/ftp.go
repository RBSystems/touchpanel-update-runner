package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func SendFTPRequest(touchpanel TouchpanelStatus, path string, file string) {
	reqInfo := FtpRequest{
		Address: touchpanel.Address,
		CallbackAddress:   os.Getenv("TOUCHPANEL_UPDATE_RUNNER_ADDRESS") + "/callbacks/afterFTP",
		Path:              path,
		File:              file,
		Identifier:        touchpanel.UUID,
	}

	b, _ := json.Marshal(&reqInfo)

	resp, err := http.Post(os.Getenv("FTP_MICROSERVICE_ADDRESS"), "application/json", bytes.NewBuffer(b))
	if err != nil {
		ReportError(touchpanel, err)
	}

	defer resp.Body.Close()
	b, _ = ioutil.ReadAll(resp.Body)

	fmt.Printf("%s Submission response: %s\n", touchpanel.Address, b)
}
