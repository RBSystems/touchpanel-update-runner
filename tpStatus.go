package main

import (
	"errors"
	"time"
)

//Struct to represent a single touchpanel.
type tpStatus struct {
	UUID              string //UUID that is assigned to each touchpanel
	RoomName          string //the name of the room associate with this touchpanel
	Type              string
	IPAddress         string    //IPAddress of the touchpanel
	Steps             []step    //List of steps in the update process.
	StartTime         time.Time //Time the update process was started
	EndTime           time.Time //Time the update process finished, or errored out.
	CurStatus         string    //The current status (Step) of the touchpanel
	IPTable           IPTable   //The IPTable associated with this touchpanel.
	FirmwareVersion   string    //The version of the firmware loaded on the touchpanel
	ProjectDate       string    //The compile date of the project loaded on the device.
	Information       modelInformation
	statusInformation []string
	errorInfo         []string
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
