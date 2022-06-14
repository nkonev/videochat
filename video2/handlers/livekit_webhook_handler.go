package handlers

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
	lksdk "github.com/livekit/server-sdk-go"
	"nkonev.name/video/config"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
)

type LivekitWebhookHandler struct {
	config              *config.ExtendedConfig
	notificationService *services.NotificationService
	livekitRoomClient   *lksdk.RoomServiceClient
}

func NewLivekitWebhookHandler(config *config.ExtendedConfig, notificationService *services.NotificationService, livekitRoomClient *lksdk.RoomServiceClient) *LivekitWebhookHandler {
	return &LivekitWebhookHandler{
		config:              config,
		notificationService: notificationService,
		livekitRoomClient:   livekitRoomClient,
	}
}

func (h *LivekitWebhookHandler) GetLivekitWebhookHandler() echo.HandlerFunc {
	livekitConfig := h.config.LivekitConfig
	return func(c echo.Context) error {
		authProvider := auth.NewSimpleKeyProvider(
			livekitConfig.Api.Key, livekitConfig.Api.Secret,
		)

		// event is a livekit.WebhookEvent{} object
		event, err := webhook.ReceiveWebhookEvent(c.Request(), authProvider)
		if err != nil {
			// could not validate, handle error
			Logger.Errorf("got error during webhook.ReceiveWebhookEvent %v %v", event, err)
			return err
		}

		// consume WebhookEvent
		Logger.Debugf("got %v", event)

		if event.Event == "participant_joined" || event.Event == "participant_left" {
			participant := event.Participant
			userId, err := utils.ParseInt64(participant.Identity)
			if err != nil {
				Logger.Errorf("got error during parsing userId from event=%v, %v", event, err)
				goto exit
			}
			md := &MetadataDto{}
			err = json.Unmarshal([]byte(participant.Metadata), md)
			if err != nil {
				Logger.Errorf("got error during parsing metadata from event=%v, %v", event, err)
				goto exit
			}

			var mutedTracks = map[string]dto.MuteInfo{}

			// TODO right after connection this contains no elements
			for _, track := range participant.GetTracks() {
				mutedTracks[track.Sid] = dto.MuteInfo{
					Kind:  track.Type.String(),
					Muted: track.Muted,
				}
			}

			notificationDto := &dto.NotifyDto{
				UserId:      userId,
				Login:       md.Login,
				MutedTracks: mutedTracks,
			}

			chatId, err := utils.GetRoomIdFromName(event.Room.Name)
			if err != nil {
				Logger.Error(err, "error during reading chat id from room name event=%v, %v", event.Room.Name)
				goto exit
			}

			var req *livekit.ListParticipantsRequest = &livekit.ListParticipantsRequest{Room: event.Room.Name}
			participants, err := h.livekitRoomClient.ListParticipants(c.Request().Context(), req)
			if err != nil {
				Logger.Errorf("got error during getting participants from livekit event=%v, %v", event, err)
				goto exit
			}

			var usersCount = int64(len(participants.Participants))

			Logger.Infof("Sending notificationDto userId=%v, chatId=%v", userId, chatId)
			err = h.notificationService.Notify(chatId, usersCount, notificationDto)
			if err != nil {
				Logger.Errorf("got error during notificationService.Notify event=%v, %v", event, err)
				goto exit
			}
		}
	exit:

		return nil
	}
}
