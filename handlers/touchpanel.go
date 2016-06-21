package handlers

import (
	"net/http"
	"time"

	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/labstack/echo"
)

func GetAllTouchpanelStatus(context echo.Context) error {
	var info []helpers.TouchpanelStatus

	if len(helpers.TouchpanelStatusMap) == 0 {
		return context.JSON(http.StatusOK, "No touchpanels currently updating")
	}

	for _, v := range helpers.TouchpanelStatusMap {
		info = append(info, v)
	}

	return context.JSON(http.StatusOK, info)
}

func GetAllTouchpanelStatusConcise(context echo.Context) error {
	var info []string
	info = append(info, "IP\tStatus\tError\n")

	if len(helpers.TouchpanelStatusMap) == 0 {
		return context.JSON(http.StatusOK, "No touchpanels currently updating")
	}

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

func GetTouchpanelStatus(context echo.Context) error {
	ip := context.Param("address")

	var toReturn []helpers.TouchpanelStatus

	for _, v := range helpers.TouchpanelStatusMap {
		if v.Address == ip {
			toReturn = append(toReturn, v)
		}
	}

	return context.JSON(http.StatusOK, toReturn)
}

func BuildHandlerStartTouchpanelUpdate(submissionChannel chan<- helpers.TouchpanelStatus) func(context echo.Context) error {
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

func BuildHandlerStartMultipleTPUpdate(submissionChannel chan<- helpers.TouchpanelStatus) func(context echo.Context) error {
	return func(context echo.Context) error {
		info := helpers.MultiJobInformation{}
		context.Bind(&info)

		// TODO: Check job information

		batch := time.Now().Format(time.RFC3339)

		touchpanelList := []helpers.TouchpanelStatus{}
		for i := range info.Info {
			if info.Info[i].Address == "" {
				touchpanelList = append(touchpanelList, helpers.TouchpanelStatus{
					CurrentStatus: "Could not start, no IP Address provided.",
					ErrorInfo:     []string{"No IP Address provided."}})
				continue
			}

			info.Info[i].HDConfiguration = info.HDConfiguration
			info.Info[i].TecLiteConfiguraiton = info.TecLiteConfiguraiton
			info.Info[i].FliptopConfiguration = info.FliptopConfiguration
			info.Info[i].Batch = batch

			touchpanel := helpers.StartTP(info.Info[i])

			touchpanelList = append(touchpanelList, touchpanel)
		}

		return context.JSON(http.StatusOK, touchpanelList)
	}
}
