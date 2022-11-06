package handlers

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
	"time"
)

type RecordHandler struct {
	egressClient           *lksdk.EgressClient
	restClient             *client.RestClient
	egressService          *services.EgressService
	onlyRoleAdminRecording bool
}

func NewRecordHandler(egressClient *lksdk.EgressClient, restClient *client.RestClient, egressService *services.EgressService) *RecordHandler {
	return &RecordHandler{egressClient: egressClient, restClient: restClient, egressService: egressService, onlyRoleAdminRecording: viper.GetBool("only-role-admin-recording")}
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
	if ok, err := rh.restClient.IsAdmin(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else {
		if !ok {
			return c.NoContent(http.StatusUnauthorized)
		}
	}
	if rh.onlyRoleAdminRecording && !userPrincipalDto.HasRole("ROLE_ADMIN") {
		GetLogEntry(c.Request().Context()).Errorf("Only admin car record with this configuration")
		return c.NoContent(http.StatusUnauthorized)
	}

	roomName := utils.GetRoomNameFromId(chatId)
	fileName := fmt.Sprintf("recording_%v.mp4", time.Now().Unix())
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
	streamRequest := &livekit.RoomCompositeEgressRequest{
		RoomName: roomName,
		Layout:   "speaker-dark",
		Output: &livekit.RoomCompositeEgressRequest_File{
			File: &livekit.EncodedFileOutput{
				FileType:        livekit.EncodedFileType_MP4,
				Filepath:        s3.Filepath,
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
	if ok, err := rh.restClient.IsAdmin(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
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
	if ok, err := rh.restClient.IsAdmin(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
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

	var normalCanRecord bool = true
	if rh.onlyRoleAdminRecording && !userPrincipalDto.HasRole("ROLE_ADMIN") {
		normalCanRecord = false
	}

	return c.JSON(http.StatusOK, StatusResponse{
		RecordInProcess: recordInProgress,
		CanMakeRecord:   normalCanRecord,
	})
}
