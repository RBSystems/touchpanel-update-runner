package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zenazn/goji/web"
)

func GetAllTPStatus(c web.C, w http.ResponseWriter, r *http.Request) {
	var info []TouchpanelStatus

	for _, v := range TouchpanelStatusMap {
		info = append(info, v)
	}

	b, _ := json.Marshal(&info)

	w.Header().Add("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", string(b))
}

func GetAllTPStatusConcise(c web.C, w http.ResponseWriter, r *http.Request) {
	var info []string
	info = append(info, "IP\tStatus\tError\n")

	for _, v := range TouchpanelStatusMap {
		if len(v.ErrorInfo) > 0 {
			str := v.IPAddress + "\t" + v.CurrentStatus + "\t" + v.ErrorInfo[0]
			info = append(info, str)
		} else {
			str := v.IPAddress + "\t" + v.CurrentStatus + "\t" + ""
			info = append(info, str)
		}

	}

	b, _ := json.Marshal(&info)

	w.Header().Add("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", string(b))
}
