package helpers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nu7hatch/gouuid"
	"github.com/zenazn/goji/web"
)

func BuildTouchpanel(jobInfo jobInformation) tpStatus {
	tp := tpStatus{
		IPAddress:     jobInfo.IPAddress,
		Steps:         GetTouchpanelSteps(),
		StartTime:     time.Now(),
		Force:         jobInfo.Force,
		Type:          jobInfo.Type[0],
		Batch:         jobInfo.Batch, // batch is for uploading to elastic search
		CurrentStatus: "Submitted",
	}

	// get the Information from the API about the current firmware/Project date

	// Temporary fix - assume everything is Tec HD
	tp.Information = jobInfo.HDConfiguration

	UUID, _ := uuid.NewV5(uuid.NamespaceURL, []byte("avengineers.byu.edu"+tp.IPAddress+tp.RoomName))
	tp.UUID = UUID.String()

	return tp
}

func GetTPStatus(c web.C, w http.ResponseWriter, r *http.Request) {
	ip := c.URLParams["ipAddress"]

	var toReturn []tpStatus

	for _, v := range tpStatusMap {
		if v.IPAddress == ip {
			toReturn = append(toReturn, v)
		}
	}

	b, err := json.Marshal(toReturn)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "ERROR: %s\n", err.Error())
	}
	w.Header().Add("content-type", "application/json")
	fmt.Fprintf(w, "%s", string(b))
}
