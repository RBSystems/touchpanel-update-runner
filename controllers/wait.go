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

func PostWait(c echo.Context) error {
	wr := helpers.WaitRequest{}
	c.Bind(&wr)

	fmt.Printf("%s Done Waiting\n", wr.Address)
	curTP := helpers.TouchpanelStatusMap[wr.Identifier]

	if curTP.UUID == "" {
		fmt.Printf("%s UUID not in map\n", wr.Address)
	}

	stepIndx, err := curTP.GetCurrentStep()
	if err != nil { // if we're already done
		fmt.Printf("%s Already done error %s\n", wr.Address, err.Error())
		// go ReportCompletion(curTP)
		return c.JSON(http.StatusBadRequest, err)
	}

	b, _ := json.Marshal(&wr)
	curTP.Steps[stepIndx].Info = string(b) + "\n" + curTP.Steps[stepIndx].Info // save the information about the wait into the step

	fmt.Printf("%s Wait status %s\n", wr.Address, wr.Status)

	if !strings.EqualFold(wr.Status, "success") { // If we timed out
		curTP.CurrentStatus = "Error"
		fmt.Printf("%s Error %s\n", wr.Address, wr.Status)
		helpers.ReportError(curTP, errors.New("Problem waiting for restart"))
		return c.JSON(http.StatusBadRequest, "Problem waiting for restart")
	}

	helpers.EvaluateNextStep(curTP) // get the next step

	return c.JSON(http.StatusOK, "Done")
}
