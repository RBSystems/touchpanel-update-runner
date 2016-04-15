package main

//Finish implementing this.
func getIPFromRoomName(roomName string) (string, error) {
	return "10.6.36.54", nil
}

//Sample, we'll get the information from the API later
func getTouchpanelsFromRoom(roomName string) ([]tpStatus, error) {
	var touchpanels []tpStatus

	samplePanel := tpStatus{
		Type:            "TecHD",
		RoomName:        roomName,
		IPAddress:       "10.6.36.54",
		FirmwareVersion: "4.004.0023",
		ProjectDate:     "4/2/2016 2:02:00 PM",
		Steps:           getSteps()}

	touchpanels = append(touchpanels, samplePanel)

	return touchpanels, nil
}

func getSteps() []step {
	var steps []step

	names := getStepNames()

	for indx := range names {
		steps = append(steps, step{StepName: names[indx], Completed: false})
	}

	return steps
}

func getStepNames() []string {
	var n []string
	n = append(n, "CheckCurrentVersion/Date", "Get IPTable", "Remove Old Firmware",
		"Initialzie",
		"Restart",
		"Wait",
		"Copy Firmware",
		"Update Firmware",
		"Wait",
		"Wait for Initialization",
		"Copy Project",
		"Move Project",
		"Load Project",
		"Wait",
		"Reload IPTable",
		"Validate")
	return n
}
