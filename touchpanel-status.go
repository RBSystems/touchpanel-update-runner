package main

import (
	"errors"
	"time"
)

//Struct to represent a single touchpanel.
type tpStatus struct {
	CurrentStatus   string //The current status (Step) of the touchpanel
	Hostname        string
	UUID            string //UUID that is assigned to each touchpanel
	RoomName        string //the name of the room associate with this touchpanel
	Type            string
	IPAddress       string    //IPAddress of the touchpanel
	StartTime       time.Time //Time the update process was started
	EndTime         time.Time //Time the update process finished, or errored out.
	IPTable         IPTable   //The IPTable associated with this touchpanel.
	FirmwareVersion string    //The version of the firmware loaded on the touchpanel
	ProjectDate     string    //The compile date of the project loaded on the device.
	Information     modelInformation
	Batch           string //Batch for uploading to Elastic Search.
	Attempts        int  //number of times to attempt the update.
	Force           bool //optional flag to bypass the validation and force the update.
	ErrorInfo       []string
	Steps           []step //List of steps in the update process.
}

//GetCurStatus gets the current step (the first item in the list of steps that isn't completed).
//returns it's name/location in array. Returns an error if completed.
func (t *tpStatus) GetCurStep() (int, error) {
	//fmt.Printf("Steps: %v\n", t.Steps)
	for k := range t.Steps {
		if t.Steps[k].Completed == false {
			return k, nil
		}
	}

	return 0, errors.New("Complete")
}
