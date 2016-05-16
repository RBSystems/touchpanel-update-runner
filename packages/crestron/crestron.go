package crestron

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func Initialize(ipAddress string) error {
	fmt.Printf("%s Intializing\n", ipAddress)
	req := telnetRequest{IPAddress: ipAddress, Command: "initialize", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS")+"Confirm", "application/json", bytes.NewBuffer(bits))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	return nil
}

func RemoveOldPUF(ipAddress string) error {
	req := telnetRequest{IPAddress: ipAddress, Command: "cd /ROMDISK/user/sytem\nerase *.puf", Prompt: "TSW-750>"}
	bits, _ := json.Marshal(req)

	resp, err := http.Post(os.Getenv("TELNET_MICROSERVICE_ADDRESS"), "application/json", bytes.NewBuffer(bits))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}
