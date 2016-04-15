package main

import (
	"time"

	"github.com/nu7hatch/gouuid"
)

type configurationData struct {
	HDConfiguration      modelInformation //The information for the HDTec panels
	TecLiteConfiguraiton modelInformation //the information for the TecLite panels
	FliptopConfiguration modelInformation //The information for the fliptop panels.
	WaitTimeout          int              //the amount of time to wait for each touchpanel to come back after a reboot. Defaults to 300
}

//Represents information needed to update the touchpanels.
type modelInformation struct {
	FirmwareLocation string //The location of the .puf file to be loaded.
	ProjectLocation  string //The locaton of the compiled project file to be loaded.
	ProjectDate      string //The compile date of the project to be loaded.
	FirmwareVersion  string //The version of the firmeware to be loaded.
}

//A struct used to monitor each room status, as each room might have more than one touchpanel
type roomProgress struct {
	UUID     uuid.UUID
	RoomName string
	TPStatus []tpStatus
	status   string
}

//Struct to represent a single touchpanel.
type tpStatus struct {
	UUID            uuid.UUID //UUID that is assigned to each touchpanel
	IPAddress       string    //IPAddress of the touchpanel
	Steps           []step    //List of steps in the update process.
	StartTime       time.Time //Time the update process was started
	EndTime         time.Time //Time the update process finished, or errored out.
	CurStatus       string    //The current status (Step) of the touchpanel
	IPTable         IPTable   //The IPTable associated with this touchpanel.
	FirmwareVersion string    //The version of the firmware loaded on the touchpanel
	ProjectDate     string    //The compile date of the project loaded on the device.
}

//Defines one step, it's completion status, as well as any information gathered from the step.
type step struct {
	StepName  string //Name of the step
	Completed bool   //if the step has been completed.
	Info      string //Any information gathered from the step. Usually the JSON body retrieved.
}

//IPTable represents an IPTable returend from a crestron device
type IPTable struct {
	Entries []IPEntry
}

//IPEntry represents a single entry in the IPTable
type IPEntry struct {
	CipID             string `json:"CIP_ID"`
	Type              string
	Status            string
	DevID             string
	Port              string
	IPAddressSitename string
}
