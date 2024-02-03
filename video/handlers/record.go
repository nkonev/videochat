package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
	"time"
)

type RecordHandler struct {
	egressClient  *lksdk.EgressClient
	restClient    *client.RestClient
	egressService *services.EgressService
	conf          *config.ExtendedConfig
	recordPreset  livekit.EncodingOptionsPreset
}

func NewRecordHandler(egressClient *lksdk.EgressClient, restClient *client.RestClient, egressService *services.EgressService, conf *config.ExtendedConfig) (*RecordHandler, error) {
	var recordPreset livekit.EncodingOptionsPreset
	switch conf.RecordPreset {
	case "H264_720P_30":
		recordPreset = livekit.EncodingOptionsPreset_H264_720P_30
	case "H264_720P_60":
		recordPreset = livekit.EncodingOptionsPreset_H264_720P_60
	case "H264_1080P_30":
		recordPreset = livekit.EncodingOptionsPreset_H264_1080P_30
	case "H264_1080P_60":
		recordPreset = livekit.EncodingOptionsPreset_H264_1080P_60
	case "PORTRAIT_H264_720P_30":
		recordPreset = livekit.EncodingOptionsPreset_PORTRAIT_H264_720P_30
	case "PORTRAIT_H264_720P_60":
		recordPreset = livekit.EncodingOptionsPreset_PORTRAIT_H264_720P_60
	case "PORTRAIT_H264_1080P_30":
		recordPreset = livekit.EncodingOptionsPreset_PORTRAIT_H264_1080P_30
	case "PORTRAIT_H264_1080P_60":
		recordPreset = livekit.EncodingOptionsPreset_PORTRAIT_H264_1080P_60

	default:
		return nil, errors.New("Unexpected value of recordPreset")
	}

	return &RecordHandler{egressClient: egressClient, restClient: restClient, egressService: egressService, conf: conf, recordPreset: recordPreset}, nil
}

func (rh *RecordHandler) canRecord(ctx context.Context, chatId int64, userPrincipalDto *auth.AuthResult) (bool, error) {
	if rh.conf.OnlyRoleAdminRecording && !userPrincipalDto.HasRole("ROLE_ADMIN") {
		return false, nil
	}
	if ok, err := rh.restClient.IsAdmin(userPrincipalDto.UserId, chatId, ctx); err != nil {
		return false, fmt.Errorf("Error during cheching is chat admin for userId %v, chatId %v", userPrincipalDto.UserId, chatId)
	} else {
		return ok, nil
	}
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
	canRecord, err := rh.canRecord(c.Request().Context(), chatId, userPrincipalDto)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking can record: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !canRecord {
		GetLogEntry(c.Request().Context()).Errorf("Only admin car record with this configuration")
		return c.NoContent(http.StatusUnauthorized)
	}

	roomName := utils.GetRoomNameFromId(chatId)
	fileName := fmt.Sprintf("recording_%v.mp4", time.Now().Format("20060102150405"))
	s3, err := rh.restClient.GetS3(fileName, chatId, userPrincipalDto.UserId, c.Request().Context())
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during gettting s3 %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	s3u := livekit.EncodedFileOutput_S3{
		S3: &livekit.S3Upload{
			AccessKey:      s3.AccessKey,
			Secret:         s3.Secret,
			Region:         s3.Region,
			Endpoint:       s3.Endpoint,
			Bucket:         s3.Bucket,
			ForcePathStyle: true,
			Metadata:       s3.Metadata,
		},
	}
	preset := rh.recordPreset
	streamRequest := &livekit.RoomCompositeEgressRequest{
		RoomName: roomName,
		Layout:   "speaker-dark",
		FileOutputs: []*livekit.EncodedFileOutput{
			&livekit.EncodedFileOutput{
				FileType:        livekit.EncodedFileType_MP4,
				Filepath:        s3.Filepath,
				Output:          &s3u,
				DisableManifest: true,
			},
		},
		AudioOnly: false,
		VideoOnly: false,
		Options: &livekit.RoomCompositeEgressRequest_Preset{
			Preset: preset,
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
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("id"))
	if err != nil {
		return err
	}
	canRecord, err := rh.canRecord(c.Request().Context(), chatId, userPrincipalDto)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking can record: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !canRecord {
		GetLogEntry(c.Request().Context()).Errorf("Only admin car record with this configuration")
		return c.NoContent(http.StatusUnauthorized)
	}

	egresses, err := rh.egressService.GetActiveEgresses(chatId, c.Request().Context())
	if err != nil {
		return err
	}

	for egressId, ownerId := range egresses {
		if ownerId == userPrincipalDto.UserId {
			_, err := rh.egressClient.StopEgress(c.Request().Context(), &livekit.StopEgressRequest{EgressId: egressId})
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error during stoppping recording %v", err)
			}
		} else {
			GetLogEntry(c.Request().Context()).Warnf("Attempt to stop not own egress %v", egressId)
		}
	}

	return c.NoContent(http.StatusAccepted)
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

	canRecord, err := rh.canRecord(c.Request().Context(), chatId, userPrincipalDto)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking can record: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if !canRecord {
		return c.JSON(http.StatusOK, StatusResponse{
			RecordInProcess: false,
			CanMakeRecord:   false,
		})
	}

	egresses, err := rh.egressService.GetActiveEgresses(chatId, c.Request().Context())
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during get active egresses: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	recordInProgress := false
	for _, ownerId := range egresses {
		if ownerId == userPrincipalDto.UserId {
			recordInProgress = true
		}
	}

	return c.JSON(http.StatusOK, StatusResponse{
		RecordInProcess: recordInProgress,
		CanMakeRecord:   canRecord,
	})
}
