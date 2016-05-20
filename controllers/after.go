package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/labstack/echo"
)

func AfterFTPHandle(c echo.Context) error {
	fr := helpers.FtpRequest{}
	c.Bind(&fr)

	curTP := helpers.TouchpanelStatusMap[fr.Identifier]

	fmt.Printf("%s Back from FTP\n", curTP.IPAddress)
	stepIndx, err := curTP.GetCurrentStep()
	if err != nil { // if we're already done
		// go ReportCompletion(curTP)
		return c.JSON(http.StatusBadRequest, "Error")
	}

	// PROBLEM: I'm not sure what's supposed to be saved here
	curTP.Steps[stepIndx].Info = "Poots" // save the information about the wait into the step

	if !strings.EqualFold(fr.Status, "success") { // If we timed out
		fmt.Printf("%s Error: %s \n %s \n", fr.IPAddressHostname, fr.Status, fr.Error)
		curTP.CurrentStatus = "Error"
		helpers.ReportError(curTP, errors.New("Problem waiting for restart"))
		return c.JSON(http.StatusBadRequest, "Problem waiting for restart")
	}

	helpers.StartWait(curTP)

	return c.JSON(http.StatusOK, "Done")
}
