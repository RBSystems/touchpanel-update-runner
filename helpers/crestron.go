package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func Initialize(ipAddress string) error {
	fmt.Printf("%s Intializing\n", ipAddress)
	req := TelnetRequest{IPAddress: ipAddress, Command: "initialize", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS")+"Confirm", "application/json", bytes.NewBuffer(bits))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	// Wait for it to come back from initialize
	err = startWait(tp)
	if err != nil {
		// TODO: Decide what to do here
		fmt.Printf("%s ERROR: %s\n", tp.IPAddress, err.Error())
		reportError(tp, err)
		return
	}

	return nil
}

func RemoveOldPUF(ipAddress string) error {
	req := TelnetRequest{IPAddress: ipAddress, Command: "cd /ROMDISK/user/sytem\nerase *.puf", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS"), "application/json", bytes.NewBuffer(bits))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}
