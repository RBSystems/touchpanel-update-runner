package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/zenazn/goji/web"
)

func SendFTPRequest(tp TouchpanelStatus, path string, file string) {
	reqInfo := ftpRequest{
		IPAddressHostname: tp.IPAddress,
		CallbackAddress:   os.Getenv("TOUCHPANEL_UPDATE_RUNNER_ADDRESS") + "/callbacks/afterFTP",
		Path:              path,
		File:              file,
		Identifier:        tp.UUID,
	}

	b, _ := json.Marshal(&reqInfo)

	resp, err := http.Post(os.Getenv("FTP_MICROSERVICE_ADDRESS"), "application/json", bytes.NewBuffer(b))
	if err != nil {
		ReportError(tp, err)
	}

	defer resp.Body.Close()
	b, _ = ioutil.ReadAll(resp.Body)

	fmt.Printf("%s Submission response: %s\n", tp.IPAddress, b)
}

func afterFTPHandle(c web.C, w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)

	var fr ftpRequest
	json.Unmarshal(b, &fr)

	curTP := TouchpanelStatusMap[fr.Identifier]

	fmt.Printf("%s Back from FTP\n", curTP.IPAddress)
	stepIndx, err := curTP.GetCurrentStep()
	if err != nil { // if we're already done
		// go ReportCompletion(curTP)
		return
	}

	curTP.Steps[stepIndx].Info = string(b) // save the information about the wait into the step

	if !strings.EqualFold(fr.Status, "success") { // If we timed out
		fmt.Printf("%s Error: %s \n %s \n", fr.IPAddressHostname, fr.Status, fr.Error)
		curTP.CurrentStatus = "Error"
		ReportError(curTP, errors.New("Problem waiting for restart."))
		return
	}

	startWait(curTP)
}
