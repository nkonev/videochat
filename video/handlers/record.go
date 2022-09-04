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
	streamRequest := &livekit.RoomCompositeEgressRequest{
		RoomName: roomName,
		Layout:   "speaker-dark",
		Output: &livekit.RoomCompositeEgressRequest_File{
			File: &livekit.EncodedFileOutput{
				FileType: livekit.EncodedFileType_MP4,
				Filepath: fmt.Sprintf("/files/chat/%v/recording_%v.mp4", chatId, time.Now()),
				Output:   &livekit.EncodedFileOutput_S3{},
			},
		},
	}

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
