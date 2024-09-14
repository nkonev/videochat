package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
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
	livekitRoomClient client.LivekitRoomClient
}

func NewUserHandler(chatClient *client.RestClient, userService *services.UserService, livekitRoomClient client.LivekitRoomClient) *UserHandler {
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
	if ok, err := h.chatClient.CheckAccess(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	var roomName = utils.GetRoomNameFromId(chatId)
	usersCount, _, err := h.userService.CountUsers(c.Request().Context(), roomName)
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
	if ok, err := h.chatClient.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	userId, err := utils.ParseInt64(c.QueryParam("userId"))
	if err != nil {
		return err
	}

	h.userService.KickUserHavingChatId(c.Request().Context(), chatId, userId)

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
	if ok, err := h.chatClient.IsAdmin( c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
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
		metadata, err := utils.ParseParticipantMetadataOrNull(participant)
		if err != nil {
			Logger.Errorf("got error during parsing metadata from participant=%v chatId=%v, %v", participant, chatId, err)
			continue
		}
		if metadata == nil {
			continue
		}

		if metadata.UserId == userId {
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
