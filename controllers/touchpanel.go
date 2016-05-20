package controllers

import (
	"net/http"

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
			str := v.IPAddress + "\t" + v.CurrentStatus + "\t" + v.ErrorInfo[0]
			info = append(info, str)
		} else {
			str := v.IPAddress + "\t" + v.CurrentStatus + "\t" + ""
			info = append(info, str)
		}
	}

	return c.JSON(http.StatusOK, info)
}
