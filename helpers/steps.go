package helpers

import (
	"errors"
	"fmt"
)

// Currently if this changes we need to edit the status updaters and post wait

// TODO: Find a way to make this dynamic (maybe a linked list type thing)
func getTouchpanelStepNames() []string {
	var names []string
	names = append(names,
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
	return names
}

// CompleteStep sends a complete step to the update channel
func CompleteStep(touchpanel TouchpanelStatus, step int, curStatus string) {
	touchpanel.Steps[step].Completed = true
	touchpanel.CurrentStatus = curStatus

	UpdateChannel <- touchpanel
}

func GetTouchpanelSteps() []step {
	var allSteps []step

	names := getTouchpanelStepNames()

	for i := range names {
		allSteps = append(allSteps, step{StepName: names[i], Completed: false})
	}

	return allSteps
}

// GetCurrentStep gets the current step (the first item in the list of steps that isn't completed) and returns it's name/location in array
func (touchpanelStatus *TouchpanelStatus) GetCurrentStep() (int, error) {
	// fmt.Printf("Steps: %v\n", t.Steps)
	for i := range touchpanelStatus.Steps {
		if touchpanelStatus.Steps[i].Completed == false {
			return i, nil
		}
	}

	return 0, errors.New("Complete")
}

// TODO: Move all steps (0-3) to this paradigm
func EvaluateNextStep(currentTouchpanel TouchpanelStatus) {
	// -----------------------------------------
	// DEBUG
	// -----------------------------------------
	// for i := 0; i < 7; i++ {
	// 			currentTouchpanel.Steps[i].Completed = true
	// 	}
	// -----------------------------------------
	// DEBUG
	// -----------------------------------------

	stepIndex, err := currentTouchpanel.GetCurrentStep()
	if err != nil {
		return
	}

	switch stepIndex { // determine where to go next
	case 0:
		CompleteStep(currentTouchpanel, stepIndex, "Validating")

		go RetrieveIPTable(currentTouchpanel)
	case 1:
		fmt.Printf("%s IP Table retrieved\n", currentTouchpanel.Address)
		need, str := ValidateNeedForUpdate(currentTouchpanel, false)
		if !need {
			fmt.Printf("%s Not needed: %s\n", currentTouchpanel.Address, str)
			reportNotNeeded(currentTouchpanel, "Not Needed: "+str)
			return
		}

		fmt.Printf("%s Done validating\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndex, "Removing old Firmware")
		go RemoveOldFirmware(currentTouchpanel)
	case 2:
		fmt.Printf("%s Old Firmware removed\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndex, "Initializing")

		go InitializeTouchpanel(currentTouchpanel)
	case 3: // Initialize - next is copy firmware
		fmt.Printf("%s Moving to copy firmware\n", currentTouchpanel.Address)

		CompleteStep(currentTouchpanel, stepIndex, "Sending Firmware")
		go SendFirmware(currentTouchpanel) // Ship this off concurrently - don't block
	case 4:
		fmt.Printf("%s Moving to update firmware\n", currentTouchpanel.Address)

		CompleteStep(currentTouchpanel, stepIndex, "Updating Firmware")
		go UpdateFirmware(currentTouchpanel)
	case 5:
		fmt.Printf("%s Done updating firmware\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndex, "Sending Project")

		go CopyProject(currentTouchpanel)
	case 6:
		fmt.Printf("%s Project copied\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndex, "Moving Project")

		go moveProject(currentTouchpanel)
	case 7:
		fmt.Printf("%s Project Moved\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndex, "Loading Project")

		go loadProject(currentTouchpanel)
	case 8:
		fmt.Printf("%s Project Loaded\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndex, "Reload IPTable")

		go ReloadIPTable(currentTouchpanel)
	case 9:
		fmt.Printf("%s IPTable loaded\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndex, "Validating")

		go validateTP(currentTouchpanel)
	default:
	}
}
