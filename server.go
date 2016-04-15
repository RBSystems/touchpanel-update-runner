package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nu7hatch/gouuid"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

func buildStartRoomUpdate(submissionChannel chan<- roomProgress) func(c web.C, w http.ResponseWriter, r *http.Request) {
	return func(c web.C, w http.ResponseWriter, r *http.Request) {
		roomName := c.URLParams["roomName"]
		roomToAdd := roomProgress{RoomName: roomName}

		uuid, _ := uuid.NewV5(uuid.NamespaceURL, []byte("AVmetrics1.byu.edu/updateTP/"+roomName))
		roomToAdd.UUID = *uuid
		roomToAdd.status = "submitted"

		submissionChannel <- roomToAdd

		bits, _ := json.Marshal(&roomToAdd)

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

func main() {

	//build our handlers, to have access to channels they must be wrapped
	submissionChannel := make(chan roomProgress, 50)

	startRoomUpdate := buildStartRoomUpdate(submissionChannel)

	goji.Post("/startUpdate/all", startUpdate)
	goji.Post("/startUpdate/SingleTP/:ipAddress", startTPUpdate)
	goji.Post("/startUpdate/:roomName", startRoomUpdate)
	goji.Get("/checkStatus/:roomName", checkRoomUpdate)
}
