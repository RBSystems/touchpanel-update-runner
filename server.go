package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/nu7hatch/gouuid"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

var tpStatusMap map[string]tpStatus //global map of tpStatus to allow for status updates.
var config configuration            //global configuration data, readonly.

func buildStartTPUpdate(submissionChannel chan<- tpStatus) func(c web.C, w http.ResponseWriter, r *http.Request) {
	return func(c web.C, w http.ResponseWriter, r *http.Request) {
		ipaddr := c.URLParams["ipAddress"]

		bits, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err.Error())
		}
		var jobInfo = jobInformation{}

		err = json.Unmarshal(bits, &jobInfo)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err.Error())
		}

		//TODO: Check job information

		tp := tpStatus{
			IPAddress: ipaddr,
			Steps:     getTPSteps(),
			StartTime: time.Now(),
			CurStatus: "Submitted"}

		//get the Information from the API about the current firmware/Project date

		//-----------------------
		//Temporary fix - assume everything is HD and we're getting that in from the
		//Request body.
		//-----------------------
		tp.Information = jobInfo.HDConfiguration
		//-----------------------

		UUID, _ := uuid.NewV5(uuid.NamespaceURL, []byte("Avengineers.byu.edu"+tp.IPAddress+tp.RoomName))
		tp.UUID = UUID.String()

		submissionChannel <- tp

		bits, _ = json.Marshal(tp)
		w.Header().Add("Content-Type", "applicaiton/json")

		fmt.Fprintf(w, "%s", bits)
	}
}

func startUpdate(c web.C, w http.ResponseWriter, r *http.Request) {
	//get the IP addresses of all the rooms.
	fmt.Fprintf(w, "Not implemented.")
}

func checkRoomUpdate(c web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Not implemented.")
}

func startAllTPUpdate(c web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Not implemented.")
}

func importConfig(configPath string) configuration {
	fmt.Printf("Importing the configuration information from %v\n", configPath)

	f, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	var configurationData configuration
	json.Unmarshal(f, &configurationData)

	fmt.Printf("\n%s\n", f)

	return configurationData
}

func postWait(c web.C, w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err.Error())
	}
	var wr waitRequest
	json.Unmarshal(b, &wr)

	fmt.Printf("Done Waiting %s.\n", wr.Identifier)
	curTP := tpStatusMap[wr.Identifier]

	stepIndx, err := curTP.GetCurStep()

	if err != nil { //if we're already done.
		//go ReportCompletion(curTP)
		return
	}

	b, _ = json.Marshal(&wr)
	curTP.Steps[stepIndx].Info = string(b) //save the information about the wait into the step.

	if !strings.EqualFold(wr.Status, "success") { //If we timed out.
		curTP.CurStatus = "Error"
		reportError(curTP, errors.New("Problem waiting for restart."))
		return
	}

	evaluateNextStep(curTP) //get the next step.
}

func afterFTPHandle(c web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Back from FTP\n")
	b, _ := ioutil.ReadAll(r.Body)

	var fr ftpRequest
	json.Unmarshal(b, &fr)

	curTP := tpStatusMap[fr.Identifier]

	stepIndx, err := curTP.GetCurStep()

	if err != nil { //if we're already done.
		//go ReportCompletion(curTP)
		return
	}

	curTP.Steps[stepIndx].Info = string(b) //save the information about the wait into the step.

	if !strings.EqualFold(fr.Status, "success") { //If we timed out.
		fmt.Printf("Error: %s \n %s \n", fr.Status, fr.Error)
		curTP.CurStatus = "Error"
		reportError(curTP, errors.New("Problem waiting for restart."))
		return
	}

	evaluateNextStep(curTP) //get the next step.

	fmt.Printf("Return: %s\n", b)
}

func main() {
	var ConfigFileLocation = flag.String("config", "./config.json", "The locaton of the config file.")

	tpStatusMap = make(map[string]tpStatus)

	flag.Parse()

	config = importConfig(*ConfigFileLocation)

	//Build our channels
	submissionChannel := make(chan tpStatus, 50)

	//build our handlers, to have access to channels they must be wrapped

	startTPUpdate := buildStartTPUpdate(submissionChannel)

	//Start our functions with the appropriate
	go startRun(submissionChannel, config)

	goji.Post("/touchpanels/", startAllTPUpdate)
	goji.Post("/touchpanels/:ipAddress", startTPUpdate)
	goji.Put("/touchpanels/", startAllTPUpdate)
	goji.Put("/touchpanels/:ipAddress", startTPUpdate)
	goji.Post("/callbacks/afterWait", postWait)
	goji.Post("/callbacks/afterFTP", afterFTPHandle)
	//	goji.Get("/touchpanels/:ipAddresss/status", getTPStatus)
	//	goji.Get("/touchpanels/:ipAddresss/status/", getTPStatus)

	goji.Serve()
}
