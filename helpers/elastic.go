package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func SendToElastic(touchpanel TouchpanelStatus, retry int) {
	b, _ := json.Marshal(&touchpanel)
	resp, err := http.Post(os.Getenv("ELASTICSEARCH_ADDRESS")+"/tpupdates/"+touchpanel.Batch+"/"+touchpanel.Hostname, "application/json", bytes.NewBuffer(b))
	if err != nil {
		if retry < 2 {
			fmt.Printf("%s error posting to ELK %s Trying again\n", touchpanel.Address, err.Error())
			SendToElastic(touchpanel, retry+1)
			return
		}

		fmt.Printf("%s Could not report to ELK %s \n", touchpanel.Address, err.Error())
	} else if resp.StatusCode > 299 || resp.StatusCode < 200 {
		fmt.Printf("%s Status Code: %v\n", touchpanel.Address, resp.StatusCode)
		b, _ := ioutil.ReadAll(resp.Body)

		if retry < 2 {
			fmt.Printf("%s error posting to ELK %s Trying again\n", touchpanel.Address, string(b))
			SendToElastic(touchpanel, retry+1)
			return
		}

		fmt.Printf("%s Could not report to ELK %s \n", touchpanel.Address, string(b))
		return
	}

	defer resp.Body.Close()
	fmt.Printf("%s Reported to ELK\n", touchpanel.Address)
}
