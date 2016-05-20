package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func getIPTable(address string) (IPTable, error) {
	request := TelnetRequest{Address: address, Command: "iptable"}
	iptable := IPTable{}

	// TODO: Make the prompt generic

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return IPTable{}, err
	}

	response, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS")+"/command", "application/json", bytes.NewBuffer(requestJSON))
	if err != nil {
		return IPTable{}, err
	}

	responseJSON, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return IPTable{}, err
	}

	fmt.Printf("%s\n", responseJSON)

	err = json.Unmarshal(responseJSON, &iptable)
	if err != nil {
		return IPTable{}, err
	}

	if len(iptable.Entries) < 1 {
		return IPTable{}, errors.New("There were no entries in the IP Table")
	}

	return iptable, nil
}
