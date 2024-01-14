package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/webhook"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
)

type LivekitWebhookHandler struct {
	config              *config.ExtendedConfig
	notificationService *services.NotificationService
	userService         *services.UserService
	egressService       *services.EgressService
	restClient          *client.RestClient
	rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher
}

func NewLivekitWebhookHandler(config *config.ExtendedConfig, notificationService *services.NotificationService, userService *services.UserService, egressService *services.EgressService, restClient *client.RestClient, rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher) *LivekitWebhookHandler {
	return &LivekitWebhookHandler{
		config:              config,
		notificationService: notificationService,
		userService:         userService,
		egressService:       egressService,
		restClient:          restClient,
		rabbitUserIdsPublisher: rabbitUserIdsPublisher,
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

			chatId, err := utils.GetRoomIdFromName(event.Room.Name)
			if err != nil {
				Logger.Error(err, "error during reading chat id from room name event=%v, %v", event, event.Room.Name)
				return nil
			}

			usersCount := int64(event.Room.NumParticipants)

			participantIds, err := h.restClient.GetChatParticipantIds(chatId, c.Request().Context())
			if err != nil {
				Logger.Error(err, "Failed during getting chat participantIds")
				return err
			}

			err = h.notificationService.NotifyVideoUserCountChanged(participantIds, chatId, usersCount, c.Request().Context())
			if err != nil {
				Logger.Errorf("got error during notificationService.NotifyVideoUserCountChanged event=%v, %v", event, err)
				return nil
			}

			if event.Event == "participant_joined" {
				metadata, err := utils.ParseParticipantMetadataOrNull(event.Participant)
				if err != nil {
					Logger.Errorf("got error during parsing metadata from participant=%v chatId=%v, %v", event.Participant, chatId, err)
				} else if metadata != nil {
					var dtos []dto.VideoCallUserCallStatusChangedDto = []dto.VideoCallUserCallStatusChangedDto{
						dto.VideoCallUserCallStatusChangedDto{
							UserId:    metadata.UserId,
							IsInVideo:     true,
						},
					}
					err = h.rabbitUserIdsPublisher.Publish(&dto.VideoCallUsersCallStatusChangedDto{Users: dtos}, c.Request().Context())
					if err != nil {
						Logger.Errorf("Error during notifying about user is in video, userId=%v, chatId=%v, error=%v", metadata.UserId, chatId, err)
					}
				}
			}
		} else if event.Event == "egress_started" {

			chatId, err := utils.GetRoomIdFromName(event.EgressInfo.RoomName)
			if err != nil {
				Logger.Error(err, "error during reading chat id from room name event=%v, %v", event, event.Room.Name)
				return nil
			}

			err = h.notificationService.NotifyRecordingChanged(chatId, true, c.Request().Context())
			if err != nil {
				Logger.Errorf("got error during notificationService.NotifyRecordingChanged, %v", err)
			}
		} else if event.Event == "egress_ended" {

			chatId, err := utils.GetRoomIdFromName(event.EgressInfo.RoomName)
			if err != nil {
				Logger.Error(err, "error during reading chat id from room name event=%v, %v", event, event.Room.Name)
				return nil
			}

			err = h.notificationService.NotifyRecordingChanged(chatId, false, c.Request().Context())
			if err != nil {
				Logger.Errorf("got error during notificationService.NotifyRecordingChanged, %v", err)
			}
		}

		return nil
	}
}
