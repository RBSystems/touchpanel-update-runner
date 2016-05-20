package helpers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/zenazn/goji/web"
)

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
		ValidationChannel <- tp

		go validateFunction(tp, 0)
	}
}

func ValidateHelper() {
	for true {
		toAdd := <-ValidationChannel
		ValidationStatus[toAdd.IPAddress] = toAdd
	}
}

func validateFunction(tp TouchpanelStatus, retries int) {
	need, str := ValidateNeed(tp, true)
	hostname, _ := SendCommand(tp, "hostname", true)

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
		ValidationChannel <- tp
	} else {
		fmt.Printf("%s Not needed.", tp.IPAddress)
		tp.CurrentStatus = "Up to date."
		ValidationChannel <- tp
	}
}
