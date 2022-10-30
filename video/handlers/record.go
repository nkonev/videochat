package handlers

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
	"time"
)

type RecordHandler struct {
	egressClient  *lksdk.EgressClient
	chatClient    *client.RestClient
	egressService *services.EgressService
}

func NewRecordHandler(egressClient *lksdk.EgressClient, chatClient *client.RestClient, egressService *services.EgressService) *RecordHandler {
	return &RecordHandler{egressClient: egressClient, chatClient: chatClient, egressService: egressService}
}

func (rh *RecordHandler) StartRecording(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("id"))
	if err != nil {
		return err
	}
	if ok, err := rh.chatClient.IsAdmin(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else {
		if !ok {
			return c.NoContent(http.StatusUnauthorized)
		}
	}

	roomName := fmt.Sprintf("chat%v", chatId)
	fileUuid := uuid.New().String()
	fileItemUuid := uuid.New().String()
	filename := fmt.Sprintf("/chat/%v/%v/%v%v", chatId, fileItemUuid, fileUuid, ".mp4")

	flnm := fmt.Sprintf("recording_%v.mp4", time.Now().Unix())
	mtd := map[string]string{}
	mtd["filename"] = flnm
	mtd["ownerid"] = utils.Int64ToString(userPrincipalDto.UserId)
	mtd["chatid"] = utils.Int64ToString(chatId)
	s3u := livekit.EncodedFileOutput_S3{
		S3: &livekit.S3Upload{
			AccessKey:      "AKIAIOSFODNN7EXAMPLE",
			Secret:         "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			Region:         "europe-east",
			Endpoint:       "http://minio:9000",
			Bucket:         "files",
			ForcePathStyle: true,
			Metadata:       mtd,
		},
	}
	streamRequest := &livekit.RoomCompositeEgressRequest{
		RoomName: roomName,
		Layout:   "speaker-dark",
		Output: &livekit.RoomCompositeEgressRequest_File{
			File: &livekit.EncodedFileOutput{
				FileType:        livekit.EncodedFileType_MP4,
				Filepath:        filename,
				Output:          &s3u,
				DisableManifest: true,
			},
		},
		AudioOnly: false,
		VideoOnly: false,
		Options: &livekit.RoomCompositeEgressRequest_Preset{
			Preset: livekit.EncodingOptionsPreset_H264_720P_30,
		},
	}

	info, err := rh.egressClient.StartRoomCompositeEgress(c.Request().Context(), streamRequest)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during starting recording %v", err)
		return err
	}
	egressId := info.EgressId
	GetLogEntry(c.Request().Context()).Infof("Starting recording %v", egressId)
	return c.JSON(http.StatusAccepted, utils.H{"egressId": egressId, "recordInProcess": true})
}

func (rh *RecordHandler) StopRecording(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("id"))
	if err != nil {
		return err
	}
	if ok, err := rh.chatClient.IsAdmin(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else {
		if !ok {
			return c.NoContent(http.StatusUnauthorized)
		}
	}

	egresses, err := rh.egressService.GetActiveEgresses(chatId, c.Request().Context())
	if err != nil {
		return err
	}

	for _, egress := range egresses {
		_, err := rh.egressClient.StopEgress(c.Request().Context(), &livekit.StopEgressRequest{EgressId: egress})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during stoppping recording %v", err)
		}
	}

	return c.JSON(http.StatusAccepted, utils.H{"recordInProcess": false})
}

type StatusResponse struct {
	RecordInProcess bool `json:"recordInProcess"`
	CanMakeRecord   bool `json:"canMakeRecord"`
}

func (rh *RecordHandler) StatusRecording(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("id"))
	if err != nil {
		return err
	}
	if ok, err := rh.chatClient.IsAdmin(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else {
		if !ok {
			return c.JSON(http.StatusOK, StatusResponse{
				RecordInProcess: false,
				CanMakeRecord:   false,
			})
		}
	}

	recordInProgress, err := rh.egressService.HasActiveEgresses(chatId, c.Request().Context())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, StatusResponse{
		RecordInProcess: recordInProgress,
		CanMakeRecord:   true,
	})
}
