package helpers

import (
	"fmt"
	"strings"
	"time"
)

func ValidateFunction(tp TouchpanelStatus, retries int) {
	need, str := ValidateNeed(tp, true)
	hostname, err := SendTelnetCommand(tp, "hostname", true)
	if err != nil {
		return
	}

	if hostname != "" {
		hostname = strings.Split(hostname, ":")[1]

		if hostname != "" {
			tp.Hostname = strings.TrimSpace(hostname)
		} else {
			if retries < 2 {
				fmt.Printf("%s retrying in 30 seconds...", tp.IPAddress)
				time.Sleep(30 * time.Second)
				ValidateFunction(tp, retries+1) // Retry
				return
			}
		}
	}

	if need {
		fmt.Printf("%s needed.", tp.IPAddress)
		tp.CurrentStatus = "Needed: " + str
		ValidationChannel <- tp
	} else {
		fmt.Printf("%s Not needed.", tp.IPAddress)
		tp.CurrentStatus = "Up to date"
		ValidationChannel <- tp
	}
}
