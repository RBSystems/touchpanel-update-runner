package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/nu7hatch/gouuid"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

func buildStartRoomUpdate(submissionChannel chan<- tpStatus) func(c web.C, w http.ResponseWriter, r *http.Request) {
	return func(c web.C, w http.ResponseWriter, r *http.Request) {
		roomName := c.URLParams["roomName"]

		touchpanels, err := getTouchpanelsFromRoom(roomName) //get the touchpanels from the roomName

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Could not get touchpanels from Roomname\n")
		}

		for indx := range touchpanels {
			tp := &touchpanels[indx]

			tp.StartTime = time.Now()
			UUID, _ := uuid.NewV5(uuid.NamespaceURL, []byte("Avengineers.byu.edu"+tp.IPAddress+tp.RoomName))

			tp.UUID = UUID.String()
			tp.CurStatus = "Submitted"

			submissionChannel <- *tp
		}

		bits, _ := json.Marshal(touchpanels)
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

func startTPUpdate(c web.C, w http.ResponseWriter, r *http.Request) {
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

func main() {
	var ConfigFileLocation = flag.String("config", "./config.json", "The locaton of the config file.")

	flag.Parse()

	config := importConfig(*ConfigFileLocation)

	//Build our channels
	submissionChannel := make(chan tpStatus, 50)

	//build our handlers, to have access to channels they must be wrapped

	startRoomUpdate := buildStartRoomUpdate(submissionChannel)

	//Start our functions with the appropriate
	go run(submissionChannel, config)

	goji.Post("/startUpdate/all", startUpdate)
	goji.Post("/startUpdate/SingleTP/:ipAddress", startTPUpdate)
	goji.Post("/startUpdate/:roomName", startRoomUpdate)
	//goji.Post("/returnFromWait", returnFromWaitHandler)
	//goji.POst("/returnFromFTP", returnFromFTPHandler)
	goji.Get("/checkStatus/:roomName", checkRoomUpdate)

	goji.Serve()
}
