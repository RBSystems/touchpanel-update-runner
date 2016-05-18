package controllers

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/zenazn/goji/web"
)

func GetValidationStatus(c web.C, w http.ResponseWriter, r *http.Request) {
	hostnames := []string{}
	values := make(map[string]helpers.TouchpanelStatus)

	for _, v := range helpers.ValidationStatus {
		values[v.Hostname] = v
		hostnames = append(hostnames, v.Hostname)
	}

	sort.Strings(hostnames)
	fmt.Printf("Length of Map: %v\n", len(helpers.ValidationStatus))
	fmt.Printf("Length of hostnames: %v\n", len(hostnames))

	w.Header().Add("Content-Type", "text/html")

	for h := range hostnames {
		cur := values[hostnames[h]]
		fmt.Fprintf(w, "%s \t\t\t %s \t\t\t %s \n", cur.Hostname, cur.IPAddress, cur.CurrentStatus)
	}
}
