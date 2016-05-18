package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func SendFTPRequest(tp TouchpanelStatus, path string, file string) {
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
