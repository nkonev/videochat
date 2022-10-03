package handlers

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
	"net/http"
	. "nkonev.name/video/logger"
	"nkonev.name/video/utils"
	"time"
)

type RecordHandler struct {
	egressClient *lksdk.EgressClient
}

func NewRecordHandler(egressClient *lksdk.EgressClient) *RecordHandler {
	return &RecordHandler{egressClient: egressClient}
}

func (rh *RecordHandler) StartRecording(c echo.Context) error {
	chatId, err := utils.ParseInt64(c.Param("id"))
	if err != nil {
		return err
	}
	roomName := fmt.Sprintf("chat%v", chatId)
	filePath := fmt.Sprintf("/files/chat/%v/recording_%v.mp4", chatId, time.Now().Unix())
	s3u := livekit.EncodedFileOutput_S3{
		S3: &livekit.S3Upload{
			AccessKey: "AKIAIOSFODNN7EXAMPLE",
			Secret:    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			Region:    "europe-east",
			Endpoint:  "127.0.0.1:9000",
			Bucket:    "files",
		},
	}
	streamRequest := &livekit.RoomCompositeEgressRequest{
		RoomName: roomName,
		Layout:   "speaker-dark",
		Output: &livekit.RoomCompositeEgressRequest_File{
			File: &livekit.EncodedFileOutput{
				FileType: livekit.EncodedFileType_MP4,
				Filepath: filePath,
				Output:   &s3u,
			},
		},
		AudioOnly: false,
		VideoOnly: false,
		Options: &livekit.RoomCompositeEgressRequest_Preset{
			Preset: livekit.EncodingOptionsPreset_H264_720P_30,
		},
	}

	//reqString, _ := proto.Marshal(streamRequest)
	//rss := string(reqString)
	//GetLogEntry(c.Request().Context()).Infof("Generated request %v", rss)

	info, err := rh.egressClient.StartRoomCompositeEgress(c.Request().Context(), streamRequest)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during starting recording %v", err)
		return err
	}
	egressId := info.EgressId
	GetLogEntry(c.Request().Context()).Infof("Starting recording %v", egressId)
	return c.JSON(http.StatusAccepted, utils.H{"egressId": egressId})
}

func (rh *RecordHandler) StopRecording(c echo.Context) error {

	egressId := c.QueryParam("egressId")

	_, err := rh.egressClient.StopEgress(c.Request().Context(), &livekit.StopEgressRequest{EgressId: egressId})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during stoppping recording %v", err)
		return err
	}

	return c.NoContent(http.StatusAccepted)
}
