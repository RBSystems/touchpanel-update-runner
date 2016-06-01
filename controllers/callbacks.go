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

// FTPCallback is the endpoint we hit when a touchpanel comes back from the FTP microservice
func FTPCallback(context echo.Context) error {
	request := helpers.FTPRequest{}
	context.Bind(&request)

	currentTouchpanel := helpers.TouchpanelStatusMap[request.CallbackIdentifier]

	fmt.Printf("%s Back from FTP\n", currentTouchpanel.Address)
	stepIndex, err := currentTouchpanel.GetCurrentStep()
	if err != nil { // If we're already done
		// go ReportCompletion(currentTouchpanel)
		return context.JSON(http.StatusBadRequest, "Error")
	}

	// TODO: I'm not sure what's supposed to be saved here (Joe was originally saving the raw body of the request)
	currentTouchpanel.Steps[stepIndex].Info = "Poots" // Save the information about the wait into the step

	if !strings.EqualFold(request.Status, "success") { // If we timed out
		fmt.Printf("%s Error: %s \n %s \n", request.DestinationAddress, request.Status, request.Error)
		currentTouchpanel.CurrentStatus = "Error"
		errorResponse := errors.New("Problem waiting for restart")

		helpers.ReportError(currentTouchpanel, errorResponse)

		return context.JSON(http.StatusBadRequest, errorResponse.Error())
	}

	helpers.StartWait(currentTouchpanel)

	return context.JSON(http.StatusOK, "Done")
}

// WaitCallback is the endpoint we hit when a touchpanel comes back from the Wait for Reboot microservice
func WaitCallback(context echo.Context) error {
	request := helpers.WaitRequest{}
	err := context.Bind(&request)
	if err != nil {
		return err
	}

	fmt.Printf("%s Done waiting\n", request.Address)
	currentTouchpanel := helpers.TouchpanelStatusMap[request.Identifier]

	if currentTouchpanel.UUID == "" {
		fmt.Printf("%s UUID not in map\n", request.Address)
	}

	stepIndex, err := currentTouchpanel.GetCurrentStep()
	if err != nil { // If we're already done
		fmt.Printf("%s Already done\n", request.Address)
		// go ReportCompletion(currentTouchpanel)
		return context.JSON(http.StatusBadRequest, err)
	}

	requestJSON, err := json.Marshal(&request)
	if err != nil {
		return err
	}

	currentTouchpanel.Steps[stepIndex].Info = string(requestJSON) + "\n" + currentTouchpanel.Steps[stepIndex].Info // Save the information about the wait into the step

	fmt.Printf("%s Wait status: %s\n", request.Address, request.Status)

	if !strings.EqualFold(request.Status, "success") { // If we timed out
		currentTouchpanel.CurrentStatus = "Error"
		fmt.Printf("%s Error %s\n", request.Address, request.Status)
		helpers.ReportError(currentTouchpanel, errors.New("Problem waiting for restart"))
		return context.JSON(http.StatusBadRequest, "Problem waiting for restart")
	}

	err = helpers.EvaluateNextStep(currentTouchpanel) // Get the next step
	if err != nil {
		return err
	}

	return context.JSON(http.StatusOK, "Done")
}
