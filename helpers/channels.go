package helpers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo"
)

func startTP(jobInfo jobInformation) tpStatus {
	tp := BuildTouchpanel(jobInfo)
	fmt.Printf("%s Starting.\n", tp.IPAddress)
	go StartRun(tp)

	return tp
}

func BuildControllerStartTouchpanelUpdate(submissionChannel chan<- tpStatus) func(c echo.Context) error {
	return func(c echo.Context) error {
		ipaddr := c.Param("ipAddress")
		batch := time.Now().Format(time.RFC3339)

		jobInfo := jobInformation{}
		c.Bind(jobInfo)

		jobInfo.IPAddress = ipaddr
		jobInfo.Batch = batch

		// TODO: Check job information

		tp := startTP(jobInfo)

		c.JSON(http.StatusOK, tp)
	}
}

func BuildStartMultipleTPUpdate(submissionChannel chan<- tpStatus) func(c echo.Context) error {
	return func(c echo.Context) error {
		info := multiJobInformation{}
		c.Bind(&info)

		// TODO: Check job information

		batch := time.Now().Format(time.RFC3339)

		tpList := []tpStatus{}
		for j := range info.Info {
			if info.Info[j].IPAddress == "" {
				tpList = append(tpList, tpStatus{
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

		c.JSON(http.StatusOK, bits)
	}
}
