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
var updateChannel chan tpStatus

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
		jobInfo.IPAddress = ipaddr

		//TODO: Check job information

		tp := startTP(submissionChannel, jobInfo)

		bits, _ = json.Marshal(tp)
		w.Header().Add("Content-Type", "applicaiton/json")

		fmt.Fprintf(w, "%s", bits)
	}
}

func buildStartMultTPUpdate(submissionChannel chan<- tpStatus) func(c web.C, w http.ResponseWriter, r *http.Request) {
	return func(c web.C, w http.ResponseWriter, r *http.Request) {
		bits, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err.Error())
		}
		var jobInfo []jobInformation

		err = json.Unmarshal(bits, &jobInfo)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err.Error())
		}
		//TODO: Check job information

		var tpList []tpStatus
		for j := range jobInfo {
			if jobInfo[j].IPAddress == "" {
				tpList = append(tpList, tpStatus{
					CurrentStatus: "Could not start, no IP Address provided.",
					ErrorInfo:     []string{"No IP Address provided."}})
				continue
			}
			tp := startTP(submissionChannel, jobInfo[j])

			tpList = append(tpList, tp)
		}

		bits, _ = json.Marshal(tpList)
		w.Header().Add("Content-Type", "applicaiton/json")

		fmt.Fprintf(w, "%s", bits)
	}
}

//Just update, so we can get around weird issues.
func updater() {
	for true {
		tpToUpdate := <-updateChannel
		fmt.Printf("%s updating status.", tpToUpdate.IPAddress)
		tpStatusMap[tpToUpdate.UUID] = tpToUpdate
	}
}

func startTP(submissionChannel chan<- tpStatus, jobInfo jobInformation) tpStatus {

	tp := tpStatus{
		IPAddress:     jobInfo.IPAddress,
		Steps:         getTPSteps(),
		StartTime:     time.Now(),
		Type:          jobInfo.Type[0],
		CurrentStatus: "Submitted"}

	//get the Information from the API about the current firmware/Project date

	//-----------------------
	//Temporary fix - assume everything is HD and we're getting that in from the
	//Request body.
	//-----------------------
	tp.Information = jobInfo.HDConfiguration
	//-----------------------

	UUID, _ := uuid.NewV5(uuid.NamespaceURL, []byte("Avengineers.byu.edu"+tp.IPAddress+tp.RoomName))
	tp.UUID = UUID.String()

	fmt.Printf("%s Starting.\n", tp.IPAddress)

	go startRun(tp)

	return tp
}

func getTPStatus(c web.C, w http.ResponseWriter, r *http.Request) {
	ip := c.URLParams["ipAddress"]

	var toReturn []tpStatus

	for _, v := range tpStatusMap {
		if v.IPAddress == ip {
			toReturn = append(toReturn, v)
		}
	}

	b, err := json.Marshal(toReturn)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "ERROR: %s\n", err.Error())
	}
	w.Header().Add("content-type", "application/json")
	fmt.Fprintf(w, "%s", string(b))
}

func startUpdate(c web.C, w http.ResponseWriter, r *http.Request) {
	//get the IP addresses of all the rooms.
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

func getAllTPStatus(c web.C, w http.ResponseWriter, r *http.Request) {
	var info []tpStatus

	for _, v := range tpStatusMap {
		info = append(info, v)
	}

	b, _ := json.Marshal(&info)

	w.Header().Add("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", string(b))
}

func getAllTPStatusConcise(c web.C, w http.ResponseWriter, r *http.Request) {
	var info []string
	info = append(info, "IP\tStatus\tError\n")

	for _, v := range tpStatusMap {
		if len(v.ErrorInfo) > 0 {
			str := v.IPAddress + "\t" + v.CurrentStatus + "\t" + v.ErrorInfo[0]
			info = append(info, str)
		} else {
			str := v.IPAddress + "\t" + v.CurrentStatus + "\t" + ""
			info = append(info, str)
		}

	}

	b, _ := json.Marshal(&info)

	w.Header().Add("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", string(b))
}

func postWait(c web.C, w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err.Error())
	}
	var wr waitRequest
	json.Unmarshal(b, &wr)

	fmt.Printf("%s Done Waiting.\n", wr.IPAddressHostname)
	curTP := tpStatusMap[wr.Identifier]

	if curTP.UUID == "" {
		fmt.Printf("%s UUID not in map.\n", wr.IPAddressHostname)
	}

	stepIndx, err := curTP.GetCurStep()

	if err != nil { //if we're already done.
		fmt.Printf("%s Already done error %s\n", wr.IPAddressHostname, err.Error())
		//go ReportCompletion(curTP)
		return
	}

	b, _ = json.Marshal(&wr)
	curTP.Steps[stepIndx].Info = string(b) //save the information about the wait into the step.

	fmt.Printf("%s Status %s", wr.IPAddressHostname, wr.Status)

	if !strings.EqualFold(wr.Status, "success") { //If we timed out.
		curTP.CurrentStatus = "Error"
		fmt.Printf("%s Error %s\n", wr.IPAddressHostname, wr.Status)
		reportError(curTP, errors.New("Problem waiting for restart."))
		return
	}

	evaluateNextStep(curTP) //get the next step.
}

func afterFTPHandle(c web.C, w http.ResponseWriter, r *http.Request) {

	b, _ := ioutil.ReadAll(r.Body)

	var fr ftpRequest
	json.Unmarshal(b, &fr)

	curTP := tpStatusMap[fr.Identifier]

	fmt.Printf("%s Back from FTP\n", curTP.IPAddress)
	stepIndx, err := curTP.GetCurStep()

	if err != nil { //if we're already done.
		//go ReportCompletion(curTP)
		return
	}

	curTP.Steps[stepIndx].Info = string(b) //save the information about the wait into the step.

	if !strings.EqualFold(fr.Status, "success") { //If we timed out.
		fmt.Printf("%s Error: %s \n %s \n", fr.IPAddressHostname, fr.Status, fr.Error)
		curTP.CurrentStatus = "Error"
		reportError(curTP, errors.New("Problem waiting for restart."))
		return
	}

	evaluateNextStep(curTP) //get the next step.

	//fmt.Printf("%s Return: %s\n", curTP.IPAddress, b)
}

func main() {
	var ConfigFileLocation = flag.String("config", "./config.json", "The locaton of the config file.")

	tpStatusMap = make(map[string]tpStatus)

	flag.Parse()

	config = importConfig(*ConfigFileLocation)

	//Build our channels
	submissionChannel := make(chan tpStatus, 50)
	updateChannel = make(chan tpStatus, 150)

	go updater()

	//build our handlers, to have access to channels they must be wrapped

	startTPUpdate := buildStartTPUpdate(submissionChannel)

	startMultipleTPUpdate := buildStartMultTPUpdate(submissionChannel)

	goji.Post("/touchpanels/", startMultipleTPUpdate)
	goji.Post("/touchpanels/:ipAddress", startTPUpdate)
	goji.Put("/touchpanels/", startMultipleTPUpdate)
	goji.Put("/touchpanels/:ipAddress", startTPUpdate)

	goji.Post("/callbacks/afterWait", postWait)
	goji.Post("/callbacks/afterFTP", afterFTPHandle)

	goji.Get("/touchpanels/:ipAddress/status", getTPStatus)
	goji.Get("/touchpanels/:ipAddress/status/", getTPStatus)

	goji.Get("/touchpanels/status", getAllTPStatus)
	goji.Get("/touchpanels/status/Concise", getAllTPStatusConcise)

	goji.Serve()
}
