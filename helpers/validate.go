package helpers

import (
	"fmt"
	"strings"
	"time"
)

func ValidateFunction(touchpanel TouchpanelStatus, retries int) {
	need, str := ValidateNeedForUpdate(touchpanel, true)
	hostname, err := SendTelnetCommand(touchpanel, "hostname", true)
	if err != nil {
		return
	}

	if hostname != "" {
		hostname = strings.Split(hostname, ":")[1]

		if hostname != "" {
			touchpanel.Hostname = strings.TrimSpace(hostname)
		} else {
			if retries < 2 {
				fmt.Printf("%s retrying in 30 seconds...", touchpanel.Address)
				time.Sleep(30 * time.Second)
				ValidateFunction(touchpanel, retries+1) // Retry
				return
			}
		}
	}

	if need {
		fmt.Printf("%s needed.", touchpanel.Address)
		touchpanel.CurrentStatus = "Needed: " + str
		ValidationChannel <- touchpanel
	} else {
		fmt.Printf("%s Not needed.", touchpanel.Address)
		touchpanel.CurrentStatus = "Up to date"
		ValidationChannel <- touchpanel
	}
}
