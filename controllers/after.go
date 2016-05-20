package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/labstack/echo"
)

func AfterFTPHandle(context echo.Context) error {
	fr := helpers.FtpRequest{}
	context.Bind(&fr)

	currentTouchpanel := helpers.TouchpanelStatusMap[fr.Identifier]

	fmt.Printf("%s Back from FTP\n", currentTouchpanel.Address)
	stepIndx, err := currentTouchpanel.GetCurrentStep()
	if err != nil { // if we're already done
		// go ReportCompletion(currentTouchpanel)
		return context.JSON(http.StatusBadRequest, "Error")
	}

	// PROBLEM: I'm not sure what's supposed to be saved here
	currentTouchpanel.Steps[stepIndx].Info = "Poots" // save the information about the wait into the step

	if !strings.EqualFold(fr.Status, "success") { // If we timed out
		fmt.Printf("%s Error: %s \n %s \n", fr.Address, fr.Status, fr.Error)
		currentTouchpanel.CurrentStatus = "Error"
		helpers.ReportError(currentTouchpanel, errors.New("Problem waiting for restart"))
		return context.JSON(http.StatusBadRequest, "Problem waiting for restart")
	}

	helpers.StartWait(currentTouchpanel)

	return context.JSON(http.StatusOK, "Done")
}
