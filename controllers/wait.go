package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/labstack/echo"
)

func PostWait(context echo.Context) error {
	wr := helpers.WaitRequest{}
	context.Bind(&wr)

	fmt.Printf("%s Done Waiting\n", wr.Address)
	currentTouchpanel := helpers.TouchpanelStatusMap[wr.Identifier]

	if currentTouchpanel.UUID == "" {
		fmt.Printf("%s UUID not in map\n", wr.Address)
	}

	stepIndx, err := currentTouchpanel.GetCurrentStep()
	if err != nil { // if we're already done
		fmt.Printf("%s Already done error %s\n", wr.Address, err.Error())
		// go ReportCompletion(currentTouchpanel)
		return context.JSON(http.StatusBadRequest, err)
	}

	b, _ := json.Marshal(&wr)
	currentTouchpanel.Steps[stepIndx].Info = string(b) + "\n" + currentTouchpanel.Steps[stepIndx].Info // save the information about the wait into the step

	fmt.Printf("%s Wait status %s\n", wr.Address, wr.Status)

	if !strings.EqualFold(wr.Status, "success") { // If we timed out
		currentTouchpanel.CurrentStatus = "Error"
		fmt.Printf("%s Error %s\n", wr.Address, wr.Status)
		helpers.ReportError(currentTouchpanel, errors.New("Problem waiting for restart"))
		return context.JSON(http.StatusBadRequest, "Problem waiting for restart")
	}

	helpers.EvaluateNextStep(currentTouchpanel) // get the next step

	return context.JSON(http.StatusOK, "Done")
}
