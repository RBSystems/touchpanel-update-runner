package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/labstack/echo"
)

func GetAllTPStatus(c echo.Context) error {
	var info []helpers.TouchpanelStatus

	for _, v := range helpers.TouchpanelStatusMap {
		info = append(info, v)
	}

	return c.JSON(http.StatusOK, info)
}

func GetAllTPStatusConcise(c echo.Context) error {
	var info []string
	info = append(info, "IP\tStatus\tError\n")

	for _, v := range helpers.TouchpanelStatusMap {
		if len(v.ErrorInfo) > 0 {
			str := v.Address + "\t" + v.CurrentStatus + "\t" + v.ErrorInfo[0]
			info = append(info, str)
		} else {
			str := v.Address + "\t" + v.CurrentStatus + "\t" + ""
			info = append(info, str)
		}
	}

	return c.JSON(http.StatusOK, info)
}

func GetTPStatus(c echo.Context) error {
	ip := c.Param("ipAddress")

	var toReturn []helpers.TouchpanelStatus

	for _, v := range helpers.TouchpanelStatusMap {
		if v.Address == ip {
			toReturn = append(toReturn, v)
		}
	}

	return c.JSON(http.StatusOK, toReturn)
}

func BuildControllerStartTouchpanelUpdate(submissionChannel chan<- helpers.TouchpanelStatus) func(c echo.Context) error {
	return func(c echo.Context) error {
		address := c.Param("address")
		batch := time.Now().Format(time.RFC3339)

		jobInfo := helpers.JobInformation{}
		c.Bind(jobInfo)

		jobInfo.Address = address
		jobInfo.Batch = batch

		// TODO: Check job information

		touchpanel := helpers.StartTP(jobInfo)

		return c.JSON(http.StatusOK, touchpanel)
	}
}

func BuildControllerStartMultipleTPUpdate(submissionChannel chan<- helpers.TouchpanelStatus) func(c echo.Context) error {
	return func(c echo.Context) error {
		info := helpers.MultiJobInformation{}
		c.Bind(&info)

		// TODO: Check job information

		batch := time.Now().Format(time.RFC3339)

		tpList := []helpers.TouchpanelStatus{}
		for j := range info.Info {
			if info.Info[j].Address == "" {
				tpList = append(tpList, helpers.TouchpanelStatus{
					CurrentStatus: "Could not start, no IP Address provided.",
					ErrorInfo:     []string{"No IP Address provided."}})
				continue
			}

			info.Info[j].HDConfiguration = info.HDConfiguration
			info.Info[j].TecLiteConfiguraiton = info.TecLiteConfiguraiton
			info.Info[j].FliptopConfiguration = info.FliptopConfiguration
			info.Info[j].Batch = batch

			touchpanel := helpers.StartTP(info.Info[j])

			tpList = append(tpList, touchpanel)
		}

		bits, err := json.Marshal(tpList)
		if err != nil {

		}

		return c.JSON(http.StatusOK, bits)
	}
}
