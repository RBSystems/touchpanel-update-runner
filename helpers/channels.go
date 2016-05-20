package helpers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo"
)

func startTP(jobInfo jobInformation) TouchpanelStatus {
	tp := BuildTouchpanel(jobInfo)
	fmt.Printf("%s Starting.\n", tp.IPAddress)
	go StartRun(tp)

	return tp
}

// Just update, so we can get around concurrent map write issues
func ChannelUpdater() {
	for true {
		tpToUpdate := <-UpdateChannel
		TouchpanelStatusMap[tpToUpdate.UUID] = tpToUpdate
	}
}

func BuildControllerStartTouchpanelUpdate(submissionChannel chan<- TouchpanelStatus) func(c echo.Context) error {
	return func(c echo.Context) error {
		ipaddr := c.Param("ipAddress")
		batch := time.Now().Format(time.RFC3339)

		jobInfo := jobInformation{}
		c.Bind(jobInfo)

		jobInfo.IPAddress = ipaddr
		jobInfo.Batch = batch

		// TODO: Check job information

		tp := startTP(jobInfo)

		return c.JSON(http.StatusOK, tp)
	}
}

func BuildStartMultipleTPUpdate(submissionChannel chan<- TouchpanelStatus) func(c echo.Context) error {
	return func(c echo.Context) error {
		info := multiJobInformation{}
		c.Bind(&info)

		// TODO: Check job information

		batch := time.Now().Format(time.RFC3339)

		tpList := []TouchpanelStatus{}
		for j := range info.Info {
			if info.Info[j].IPAddress == "" {
				tpList = append(tpList, TouchpanelStatus{
					CurrentStatus: "Could not start, no IP Address provided.",
					ErrorInfo:     []string{"No IP Address provided."}})
				continue
			}

			info.Info[j].HDConfiguration = info.HDConfiguration
			info.Info[j].TecLiteConfiguraiton = info.TecLiteConfiguraiton
			info.Info[j].FliptopConfiguration = info.FliptopConfiguration
			info.Info[j].Batch = batch

			tp := startTP(info.Info[j])

			tpList = append(tpList, tp)
		}

		bits, err := json.Marshal(tpList)
		if err != nil {

		}

		return c.JSON(http.StatusOK, bits)
	}
}
