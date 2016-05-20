package helpers

import (
	"errors"
	"fmt"
)

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

// CompleteStep sends a complete step to the update channel
func CompleteStep(touchpanel TouchpanelStatus, step int, curStatus string) {
	touchpanel.Steps[step].Completed = true
	touchpanel.CurrentStatus = curStatus

	UpdateChannel <- touchpanel
}

func GetTouchpanelSteps() []step {
	var steps []step

	names := getTouchpanelStepNames()

	for indx := range names {
		steps = append(steps, step{StepName: names[indx], Completed: false})
	}

	return steps
}

// GetCurStatus gets the current step (the first item in the list of steps that isn't completed) and returns it's name/location in array. Returns an error if completed
func (t *TouchpanelStatus) GetCurrentStep() (int, error) {
	// fmt.Printf("Steps: %v\n", t.Steps)
	for k := range t.Steps {
		if t.Steps[k].Completed == false {
			return k, nil
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

	stepIndx, err := currentTouchpanel.GetCurrentStep()

	if err != nil {
		return
	}

	switch stepIndx { // determine where to go next
	case 0:
		CompleteStep(currentTouchpanel, stepIndx, "Validating")

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
		CompleteStep(currentTouchpanel, stepIndx, "Removing old Firmware")
		go RemoveOldFirmware(currentTouchpanel)
	case 2:
		fmt.Printf("%s Old Firmware removed\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndx, "Initializing")

		go InitializeTouchpanel(currentTouchpanel)
	case 3: // Initialize - next is copy firmware
		fmt.Printf("%s Moving to copy firmware\n", currentTouchpanel.Address)

		// Set status and update the
		CompleteStep(currentTouchpanel, stepIndx, "Sending Firmware")
		go SendFirmware(currentTouchpanel) // Ship this off concurrently - don't block
	case 4:
		fmt.Printf("%s Moving to update firmware\n", currentTouchpanel.Address)

		CompleteStep(currentTouchpanel, stepIndx, "Updating Firmware")
		go UpdateFirmware(currentTouchpanel)
	case 5:
		fmt.Printf("%s Done updating firmware\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndx, "Sending Project")

		go CopyProject(currentTouchpanel)
	case 6:
		fmt.Printf("%s Project copied\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndx, "Moving Project")

		go moveProject(currentTouchpanel)
	case 7:
		fmt.Printf("%s Project Moved\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndx, "Loading Project")

		go loadProject(currentTouchpanel)
	case 8:
		fmt.Printf("%s Project Loaded\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndx, "Reload IPTable")

		go ReloadIPTable(currentTouchpanel)
	case 9:
		fmt.Printf("%s IPTable loaded\n", currentTouchpanel.Address)
		CompleteStep(currentTouchpanel, stepIndx, "Validating")

		go validateTP(currentTouchpanel)
	default:
	}
}
