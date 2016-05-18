package helpers

// Currently if this changes we need to edit the status updaters and post wait

// TODO: Find a way to make this dynamic (maybe a linked list type thing)
func getTouchpanelStepNames() []string {
	var n []string
	n = append(n,
		"Get IPTable",              // 0
		"CheckCurrentVersion/Date", // 1
		"Remove Old Firmware",      // 2
		"Initialize",               // 3
		"Copy Firmware",            // 4
		"Update Firmware",          // 5
		"Copy Project",             // 6
		"Move Project",             // 7
		"Load Project",             // 8
		"Reload IPTable",           // 9
		"Validate")                 // 10
	return n
}

func GetTouchpanelSteps() []step {
	var steps []step

	names := getTouchpanelStepNames()

	for indx := range names {
		steps = append(steps, step{StepName: names[indx], Completed: false})
	}

	return steps
}
