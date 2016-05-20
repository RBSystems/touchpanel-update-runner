package controllers

import (
	"net/http"
	"time"

	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/labstack/echo"
)

func GetAllTouchpanelStatus(context echo.Context) error {
	var info []helpers.TouchpanelStatus

	for _, v := range helpers.TouchpanelStatusMap {
		info = append(info, v)
	}

	return context.JSON(http.StatusOK, info)
}

func GetAllTouchpanelStatusConcise(context echo.Context) error {
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

	return context.JSON(http.StatusOK, info)
}

func GetTPStatus(context echo.Context) error {
	ip := context.Param("ipAddress")

	var toReturn []helpers.TouchpanelStatus

	for _, v := range helpers.TouchpanelStatusMap {
		if v.Address == ip {
			toReturn = append(toReturn, v)
		}
	}

	return context.JSON(http.StatusOK, toReturn)
}

func BuildControllerStartTouchpanelUpdate(submissionChannel chan<- helpers.TouchpanelStatus) func(context echo.Context) error {
	return func(context echo.Context) error {
		address := context.Param("address")
		batch := time.Now().Format(time.RFC3339)

		jobInfo := helpers.JobInformation{}
		context.Bind(jobInfo)

		jobInfo.Address = address
		jobInfo.Batch = batch

		// TODO: Check job information

		touchpanel := helpers.StartTP(jobInfo)

		return context.JSON(http.StatusOK, touchpanel)
	}
}

func BuildControllerStartMultipleTPUpdate(submissionChannel chan<- helpers.TouchpanelStatus) func(context echo.Context) error {
	return func(context echo.Context) error {
		info := helpers.MultiJobInformation{}
		context.Bind(&info)

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

		return context.JSON(http.StatusOK, tpList)
	}
}
