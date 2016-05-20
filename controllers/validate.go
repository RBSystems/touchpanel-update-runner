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

func Validate(c echo.Context) error {
	info := helpers.MultiJobInformation{}
	c.Bind(&info)

	batch := time.Now().Format(time.RFC3339)

	for i := range info.Info {
		fmt.Printf("Buildling TP...")

		info.Info[i].Batch = batch
		info.Info[i].HDConfiguration = info.HDConfiguration
		tp := helpers.BuildTouchpanel(info.Info[i])
		tp.Address = strings.TrimSpace(tp.Address)
		tp.CurrentStatus = "In progress"
		tp.Hostname = "TEMP " + tp.UUID
		helpers.ValidationChannel <- tp

		go helpers.ValidateFunction(tp, 0)
	}

	return c.JSON(http.StatusOK, "Done")
}

func GetValidationStatus(c echo.Context) error {
	hostnames := []string{}
	values := make(map[string]helpers.TouchpanelStatus)

	for _, v := range helpers.ValidationStatus {
		values[v.Hostname] = v
		hostnames = append(hostnames, v.Hostname)
	}

	sort.Strings(hostnames)
	fmt.Printf("Length of Map: %v\n", len(helpers.ValidationStatus))
	fmt.Printf("Length of hostnames: %v\n", len(hostnames))

	for h := range hostnames {
		cur := values[hostnames[h]]
		fmt.Println(cur.Hostname + "   " + cur.Address + "   " + cur.CurrentStatus)
	}

	return c.JSON(http.StatusOK, "Done")
}
