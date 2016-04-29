package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/nu7hatch/gouuid"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

var tpStatusMap map[string]tpStatus //global map of tpStatus to allow for status updates.
var config configuration            //global configuration data, readonly.
var updateChannel chan tpStatus

var validationStatus map[string]tpStatus
var validationChannel chan tpStatus

func buildStartTPUpdate(submissionChannel chan<- tpStatus) func(c web.C, w http.ResponseWriter, r *http.Request) {
	return func(c web.C, w http.ResponseWriter, r *http.Request) {
		ipaddr := c.URLParams["ipAddress"]
		batch := time.Now().Format(time.RFC3339)
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
		jobInfo.Batch = batch

		//TODO: Check job information

		tp := startTP(jobInfo)

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
		var info multiJobInformation

		err = json.Unmarshal(bits, &info)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err.Error())
		}
		//TODO: Check job information

		batch := time.Now().Format(time.RFC3339)

		var tpList []tpStatus
		for j := range info.Info {
			if info.Info[j].IPAddress == "" {
				tpList = append(tpList, tpStatus{
					CurrentStatus: "Could not start, no IP Address provided.",
					ErrorInfo:     []string{"No IP Address provided."}})
				continue
			}
			info.Info[j].HDConfiguration = info.HDConfiguration
			info.Info[j].TecLiteConfiguraiton = info.TecLiteConfiguraiton
			info.Info[j].FliptopConfiguration = info.FliptopConfiguration
			info.Info[j].Batch = batch

			tp := startTP(info.Info[j])

			tpList = append(tpList, tp)
		}

		bits, _ = json.Marshal(tpList)
		w.Header().Add("Content-Type", "applicaiton/json")

		fmt.Fprintf(w, "%s", bits)
	}
}

//Just update, so we can get around concurrent map write issues.
func updater() {
	for true {
		tpToUpdate := <-updateChannel
		tpStatusMap[tpToUpdate.UUID] = tpToUpdate
	}
}

func startTP(jobInfo jobInformation) tpStatus {

	tp := buildTP(jobInfo)
	fmt.Printf("%s Starting.\n", tp.IPAddress)
	go startRun(tp)
	return tp
}

func buildTP(jobInfo jobInformation) tpStatus {
	tp := tpStatus{
		IPAddress:     jobInfo.IPAddress,
		Steps:         getTPSteps(),
		StartTime:     time.Now(),
		Force:         jobInfo.Force,
		Type:          jobInfo.Type[0],
		Batch:         jobInfo.Batch, //batch is for uploading to elastic search
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
	curTP.Steps[stepIndx].Info = string(b) + "\n" + curTP.Steps[stepIndx].Info //save the information about the wait into the step.

	fmt.Printf("%s Wait status %s\n", wr.IPAddressHostname, wr.Status)

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

	startWait(curTP, config)

	//fmt.Printf("%s Return: %s\n", curTP.IPAddress, b)
}

func test(c web.C, w http.ResponseWriter, r *http.Request) {
}

func validate(c web.C, w http.ResponseWriter, r *http.Request) {
	bits, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err.Error())
	}
	var info multiJobInformation

	err = json.Unmarshal(bits, &info)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err.Error())
	}

	batch := time.Now().Format(time.RFC3339)

	for i := range info.Info {
		fmt.Printf("Buildling TP...")

		info.Info[i].Batch = batch
		info.Info[i].HDConfiguration = info.HDConfiguration
		tp := buildTP(info.Info[i])
		tp.IPAddress = strings.TrimSpace(tp.IPAddress)
		tp.CurrentStatus = "In progress"
		tp.Hostname = "TEMP " + tp.UUID
		validationChannel <- tp
		go validateFunction(tp, 0)
	}
}

func getValidationStatus(c web.C, w http.ResponseWriter, r *http.Request) {
	hostnames := []string{}
	values := make(map[string]tpStatus)

	for _, v := range validationStatus {
		values[v.Hostname] = v
		hostnames = append(hostnames, v.Hostname)
	}

	sort.Strings(hostnames)
	fmt.Printf("Length of Map: %v\n", len(validationStatus))
	fmt.Printf("Length of hostnames: %v\n", len(hostnames))

	w.Header().Add("Content-Type", "text/html")

	for h := range hostnames {
		cur := values[hostnames[h]]
		fmt.Fprintf(w, "%s \t\t\t %s \t\t\t %s \n", cur.Hostname, cur.IPAddress, cur.CurrentStatus)
	}
}

func validateHelper() {
	for true {
		toAdd := <-validationChannel
		validationStatus[toAdd.IPAddress] = toAdd
	}
}

func validateFunction(tp tpStatus, retries int) {
	need, str := validateNeed(tp, true)
	hostname, _ := sendCommand(tp, "hostname", true)

	if hostname != "" {
		hostname = strings.Split(hostname, ":")[1]
		if hostname != "" {
			tp.Hostname = strings.TrimSpace(hostname)
		} else {
			if retries < 2 {
				fmt.Printf("%s retrying in 30 seconds...", tp.IPAddress)
				time.Sleep(30 * time.Second)
				validateFunction(tp, retries+1)
				return
			}
		}
	}
	if need {
		fmt.Printf("%s needed.", tp.IPAddress)
		tp.CurrentStatus = "Needed: " + str
		validationChannel <- tp
	} else {
		fmt.Printf("%s Not needed.", tp.IPAddress)
		tp.CurrentStatus = "Up to date."
		validationChannel <- tp
	}
}

func main() {
	var ConfigFileLocation = flag.String("config", "./config.json", "The locaton of the config file.")

	tpStatusMap = make(map[string]tpStatus)
	validationStatus = make(map[string]tpStatus)

	flag.Parse()

	config = importConfig(*ConfigFileLocation)

	//Build our channels
	submissionChannel := make(chan tpStatus, 50)
	updateChannel = make(chan tpStatus, 150)
	validationChannel = make(chan tpStatus, 150)

	go updater()
	go validateHelper()

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
	goji.Get("/touchpanels/status/", getAllTPStatus)
	goji.Get("/touchpanels/status/concise", getAllTPStatusConcise)
	goji.Get("/touchpanels/status/concise/", getAllTPStatusConcise)

	goji.Post("/touchpanels/test/", test)

	goji.Post("/validate/touchpanels", validate)
	goji.Post("/validate/touchpanels/", validate)
	goji.Get("/validate/touchpanels/status", getValidationStatus)
	goji.Get("/validate/touchpanels/status/", getValidationStatus)

	goji.Serve()
}
