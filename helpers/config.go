package helpers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Configuration struct {
	WaitTimeout                      int    // the amount of time to wait for each touchpanel to come back after a reboot. Defaults to 300
	FTPMicroserviceAddress           string // Locaitons for the microservices to be used.
	TelnetMicroserviceAddress        string
	WaitForRebootMicroserviceAddress string
	ElasticsearchAddress             string
	TouchpanelUpdateRunnerAddress    string // hostname and port of the server running the touchpanel update - to be used to format the callbacks.
	AttemptLimit                     int    // Number of times to retry a panel before reporting a failure.
}

func ImportConfiguration(configurationLocation string) Configuration {
	fmt.Printf("Importing the configuration information from %v\n", configurationLocation)

	f, err := ioutil.ReadFile(configurationLocation)
	if err != nil {
		panic(err)
	}

	configuration := Configuration{}
	json.Unmarshal(f, &configuration)

	fmt.Printf("\n%s\n", f)

	return configuration
}
