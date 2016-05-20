package controllers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/labstack/echo"
)

func Validate(context echo.Context) error {
	info := helpers.MultiJobInformation{}
	context.Bind(&info)

	batch := time.Now().Format(time.RFC3339)

	for i := range info.Info {
		fmt.Printf("Buildling TP...")

		info.Info[i].Batch = batch
		info.Info[i].HDConfiguration = info.HDConfiguration
		touchpanel := helpers.BuildTouchpanel(info.Info[i])
		touchpanel.Address = strings.TrimSpace(touchpanel.Address)
		touchpanel.CurrentStatus = "In progress"
		touchpanel.Hostname = "TEMP " + touchpanel.UUID
		helpers.ValidationChannel <- touchpanel

		go helpers.ValidateFunction(touchpanel, 0)
	}

	return context.JSON(http.StatusOK, "Done")
}

func GetValidationStatus(context echo.Context) error {
	hostnames := []string{}
	values := make(map[string]helpers.TouchpanelStatus)

	for _, v := range helpers.ValidationStatus {
		values[v.Hostname] = v
		hostnames = append(hostnames, v.Hostname)
	}

	sort.Strings(hostnames)
	fmt.Printf("Length of Map: %v\n", len(helpers.ValidationStatus))
	fmt.Printf("Length of hostnames: %v\n", len(hostnames))

	for i := range hostnames {
		cur := values[hostnames[i]]
		fmt.Println(cur.Hostname + "   " + cur.Address + "   " + cur.CurrentStatus)
	}

	return context.JSON(http.StatusOK, "Done")
}
