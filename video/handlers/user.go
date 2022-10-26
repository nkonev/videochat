package handlers

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/pkg/errors"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
)

type UserHandler struct {
	chatClient        *client.RestClient
	userService       *services.UserService
	livekitRoomClient *lksdk.RoomServiceClient
}

func NewUserHandler(chatClient *client.RestClient, userService *services.UserService, livekitRoomClient *lksdk.RoomServiceClient) *UserHandler {
	return &UserHandler{chatClient: chatClient, userService: userService, livekitRoomClient: livekitRoomClient}
}

type CountUsersResponse struct {
	UsersCount int64 `json:"usersCount"`
	ChatId     int64 `json:"chatId"`
}

func (h *UserHandler) GetVideoUsers(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.chatClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	var roomName = utils.GetRoomNameFromId(chatId)
	usersCount, err := h.userService.CountUsers(c.Request().Context(), roomName)
	if err != nil {
		Logger.Errorf("got error during getting participants from http users request, %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, CountUsersResponse{UsersCount: usersCount, ChatId: chatId})
}

func (h *UserHandler) Kick(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.chatClient.IsAdmin(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	userId, err := utils.ParseInt64(c.QueryParam("userId"))
	if err != nil {
		return err
	}

	roomName := utils.GetRoomNameFromId(chatId)

	lpr := &livekit.ListParticipantsRequest{Room: roomName}
	participants, err := h.livekitRoomClient.ListParticipants(c.Request().Context(), lpr)
	if err != nil {
		Logger.Errorf("Unable to get participants %v", err)
		return err
	}

	for _, participant := range participants.Participants {
		md := &MetadataDto{}
		err = json.Unmarshal([]byte(participant.Metadata), md)
		if err != nil {
			Logger.Errorf("got error during parsing metadata from kick userId=%v from chatId=%v, %v", userId, chatId, err)
			continue
		}
		if md.UserId == userId {
			var removeReq = &livekit.RoomParticipantIdentity{
				Room:     roomName,
				Identity: participant.Identity,
			}
			Logger.Infof("Kicking userId=%v with identity %v from chatId=%v", userId, participant.Identity, chatId)
			_, err := h.livekitRoomClient.RemoveParticipant(c.Request().Context(), removeReq)
			if err != nil {
				Logger.Errorf("got error during kicking userId=%v, %v", userId, err)
				continue
			}
		}
	}
	return c.NoContent(http.StatusOK)
}

func (h *UserHandler) Mute(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.chatClient.IsAdmin(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	userId, err := utils.ParseInt64(c.QueryParam("userId"))
	if err != nil {
		return err
	}

	roomName := utils.GetRoomNameFromId(chatId)

	lpr := &livekit.ListParticipantsRequest{Room: roomName}
	participants, err := h.livekitRoomClient.ListParticipants(c.Request().Context(), lpr)
	if err != nil {
		Logger.Errorf("Unable to get participants %v", err)
		return err
	}

	for _, participant := range participants.Participants {
		md := &MetadataDto{}
		err = json.Unmarshal([]byte(participant.Metadata), md)
		if err != nil {
			Logger.Errorf("got error during parsing metadata from kick userId=%v from chatId=%v, %v", userId, chatId, err)
			continue
		}
		if md.UserId == userId {
			Logger.Infof("Muting userId=%v with identity %v from chatId=%v", userId, participant.Identity, chatId)

			for _, track := range participant.GetTracks() {
				if track.Type == livekit.TrackType_AUDIO && !track.Muted {
					var muteReq = &livekit.MuteRoomTrackRequest{
						Room:     roomName,
						Identity: participant.Identity,
						Muted:    true,
						TrackSid: track.Sid,
					}
					_, err := h.livekitRoomClient.MutePublishedTrack(c.Request().Context(), muteReq)
					if err != nil {
						Logger.Errorf("got error during muting userId=%v, %v", userId, err)
						continue
					}
				}
			}
		}
	}
	return c.NoContent(http.StatusOK)
}
