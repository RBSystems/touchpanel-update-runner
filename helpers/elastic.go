package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func SendToElastic(tp TouchpanelStatus, retry int) {
	b, _ := json.Marshal(&tp)
	resp, err := http.Post(os.Getenv("ELASTICSEARCH_ADDRESS")+"/tpupdates/"+tp.Batch+"/"+tp.Hostname, "application/json", bytes.NewBuffer(b))

	if err != nil {
		if retry < 2 {
			fmt.Printf("%s error posting to ELK %s. Trying again.\n", tp.Address, err.Error())
			SendToElastic(tp, retry+1)
			return
		}

		fmt.Printf("%s Could not report to ELK. %s \n", tp.Address, err.Error())
	} else if resp.StatusCode > 299 || resp.StatusCode < 200 {
		fmt.Printf("%s Status Code: %v\n", tp.Address, resp.StatusCode)
		b, _ := ioutil.ReadAll(resp.Body)

		if retry < 2 {
			fmt.Printf("%s error posting to ELK %s. Trying again.\n", tp.Address, string(b))
			SendToElastic(tp, retry+1)
			return
		}

		fmt.Printf("%s Could not report to ELK. %s \n", tp.Address, string(b))
		return
	}

	defer resp.Body.Close()
	fmt.Printf("%s Reported to ELK.\n", tp.Address)
}
