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

	"github.com/byuoitav/touchpanel-update-runner/controllers"
	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/jessemillar/health"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/fasthttp"
	"github.com/labstack/echo/middleware"
	"github.com/zenazn/goji/web"
)

// Just update, so we can get around concurrent map write issues
func updater() {
	for true {
		tpToUpdate := <-UpdateChannel
		TouchpanelStatusMap[tpToUpdate.UUID] = tpToUpdate
	}
}

func startUpdate(c web.C, w http.ResponseWriter, r *http.Request) {
	// get the IP addresses of all the rooms
	fmt.Fprintf(w, "Not implemented.")
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
	curTP := TouchpanelStatusMap[wr.Identifier]

	if curTP.UUID == "" {
		fmt.Printf("%s UUID not in map.\n", wr.IPAddressHostname)
	}

	stepIndx, err := curTP.GetCurrentStep()

	if err != nil { // if we're already done
		fmt.Printf("%s Already done error %s\n", wr.IPAddressHostname, err.Error())
		// go ReportCompletion(curTP)
		return
	}

	b, _ = json.Marshal(&wr)
	curTP.Steps[stepIndx].Info = string(b) + "\n" + curTP.Steps[stepIndx].Info // save the information about the wait into the step

	fmt.Printf("%s Wait status %s\n", wr.IPAddressHostname, wr.Status)

	if !strings.EqualFold(wr.Status, "success") { // If we timed out
		curTP.CurrentStatus = "Error"
		fmt.Printf("%s Error %s\n", wr.IPAddressHostname, wr.Status)
		reportError(curTP, errors.New("Problem waiting for restart."))
		return
	}

	EvaluateNextStep(curTP) // get the next step
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
		reportError(curTP, errors.New("Problem waiting for restart."))
		return
	}

	startWait(curTP, configuration)
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
		tp := BuildTouchpanel(info.Info[i])
		tp.IPAddress = strings.TrimSpace(tp.IPAddress)
		tp.CurrentStatus = "In progress"
		tp.Hostname = "TEMP " + tp.UUID
		helpers.ValidationChannel <- tp

		go validateFunction(tp, 0)
	}
}

func validateHelper() {
	for true {
		toAdd := <-helpers.ValidationChannel
		validationStatus[toAdd.IPAddress] = toAdd
	}
}

func validateFunction(tp TouchpanelStatus, retries int) {
	need, str := validateNeed(tp, true)
	hostname, _ := helpers.SendCommand(tp, "hostname", true)

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
		helpers.ValidationChannel <- tp
	} else {
		fmt.Printf("%s Not needed.", tp.IPAddress)
		tp.CurrentStatus = "Up to date."
		helpers.ValidationChannel <- tp
	}
}

func main() {
	TouchpanelStatusMap = make(map[string]TouchpanelStatus)
	validationStatus = make(map[string]TouchpanelStatus)

	flag.Parse()

	// Build our channels
	submissionChannel := make(chan TouchpanelStatus, 50)
	helpers.UpdateChannel = make(chan TouchpanelStatus, 150)
	helpers.ValidationChannel = make(chan TouchpanelStatus, 150)

	go updater()
	go validateHelper()

	// Build our handlers--to have access to channels they must be wrapped
	startTPUpdate := BuildControllerStartTouchpanelUpdate(submissionChannel)
	startMultipleTPUpdate := buildStartMultipleTPUpdate(submissionChannel)

	port := ":8004"
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())

	e.Get("/health", health.Check)

	// Touchpanels
	e.Get("/touchpanel/status", controllers.GetAllTPStatus)
	e.Get("/touchpanel/status/concise", controllers.GetAllTPStatusConcise)
	e.Get("/touchpanel/:ipAddress/status", getTPStatus)

	e.Post("/touchpanel", startMultipleTPUpdate)
	e.Post("/touchpanel/:ipAddress", startTPUpdate)
	e.Post("/touchpanel/test", test)

	e.Put("/touchpanel", startMultipleTPUpdate)
	e.Put("/touchpanel/:ipAddress", startTPUpdate)

	// Callback
	e.Post("/callback/afterWait", postWait)
	e.Post("/callback/afterFTP", afterFTPHandle)

	// Validation
	e.Get("/validate/touchpanels/status", controllers.GetValidationStatus)

	e.Post("/validate/touchpanels", validate)

	fmt.Printf("The Touchpanel Update Runner is listening on %s\n", port)
	e.Run(fasthttp.New(port))
}
