package helpers

import (
	"time"

	"github.com/nu7hatch/gouuid"
)

func BuildTouchpanel(jobInfo jobInformation) TouchpanelStatus {
	tp := TouchpanelStatus{
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
