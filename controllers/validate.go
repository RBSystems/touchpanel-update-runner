package controllers

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/labstack/echo"
)

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
		fmt.Println(cur.Hostname + "   " + cur.IPAddress + "   " + cur.CurrentStatus)
	}

	return c.JSON(http.StatusOK, "Done")
}
