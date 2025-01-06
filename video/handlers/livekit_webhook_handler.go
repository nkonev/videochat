package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/webhook"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	"nkonev.name/video/dto"
	"nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
)

type LivekitWebhookHandler struct {
	config                 *config.ExtendedConfig
	notificationService    *services.NotificationService
	userService            *services.UserService
	egressService          *services.EgressService
	restClient             *client.RestClient
	rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher
	lgr                    *logger.Logger
}

func NewLivekitWebhookHandler(config *config.ExtendedConfig, notificationService *services.NotificationService, userService *services.UserService, egressService *services.EgressService, restClient *client.RestClient, rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher, lgr *logger.Logger) *LivekitWebhookHandler {
	return &LivekitWebhookHandler{
		config:                 config,
		notificationService:    notificationService,
		userService:            userService,
		egressService:          egressService,
		restClient:             restClient,
		rabbitUserIdsPublisher: rabbitUserIdsPublisher,
		lgr:                    lgr,
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
			h.lgr.WithTracing(c.Request().Context()).Errorf("got error during webhook.ReceiveWebhookEvent %v %v", event, err)
			return err
		}

		// consume WebhookEvent
		h.lgr.WithTracing(c.Request().Context()).Debugf("got %v", event)

		if event.Event == "participant_joined" || event.Event == "participant_left" {

			chatId, err := utils.GetRoomIdFromName(event.Room.Name)
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Error(err, "error during reading chat id from room name event=%v, %v", event, event.Room.Name)
				return nil
			}

			usersCount := int64(event.Room.NumParticipants)

			err = h.restClient.GetChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
				internalErr := h.notificationService.NotifyVideoUserCountChanged(c.Request().Context(), participantIds, chatId, usersCount)
				if internalErr != nil {
					h.lgr.WithTracing(c.Request().Context()).Errorf("got error during notificationService.NotifyVideoUserCountChanged event=%v, %v", event, internalErr)
					return nil
				} else {
					return internalErr
				}
			})
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Error(err, "Failed during getting chat participantIds")
				return err
			}

			if event.Event == "participant_joined" {
				metadata, err := utils.ParseParticipantMetadataOrNull(event.Participant)
				if err != nil {
					h.lgr.WithTracing(c.Request().Context()).Errorf("got error during parsing metadata from participant=%v chatId=%v, %v", event.Participant, chatId, err)
				} else if metadata != nil {
					err = h.rabbitUserIdsPublisher.Publish(c.Request().Context(), &dto.VideoCallUsersCallStatusChangedDto{Users: []dto.VideoCallUserCallStatusChangedDto{
						{
							UserId:    metadata.UserId,
							IsInVideo: true,
						},
					}})
					if err != nil {
						h.lgr.WithTracing(c.Request().Context()).Errorf("Error during notifying about user is in video, userId=%v, chatId=%v, error=%v", metadata.UserId, chatId, err)
					}
				}
			}
		} else if event.Event == "egress_started" {

			chatId, err := utils.GetRoomIdFromName(event.EgressInfo.RoomName)
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Error(err, "error during reading chat id from room name event=%v, %v", event, event.Room.Name)
				return nil
			}

			ownerId, err := h.egressService.GetOwnerId(c.Request().Context(), event.EgressInfo)
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Errorf("Unable to get ownerId of %v: %v", event.EgressInfo.EgressId, err)
			}
			err = h.notificationService.NotifyRecordingChanged(c.Request().Context(), chatId, map[int64]bool{ownerId: true})
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Errorf("got error during notificationService.NotifyRecordingChanged, %v", err)
			}
		} else if event.Event == "egress_ended" {

			chatId, err := utils.GetRoomIdFromName(event.EgressInfo.RoomName)
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Error(err, "error during reading chat id from room name event=%v, %v", event, event.Room.Name)
				return nil
			}
			ownerId, err := h.egressService.GetOwnerId(c.Request().Context(), event.EgressInfo)
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Errorf("Unable to get ownerId of %v: %v", event.EgressInfo.EgressId, err)
			}
			err = h.notificationService.NotifyRecordingChanged(c.Request().Context(), chatId, map[int64]bool{ownerId: false})
			if err != nil {
				h.lgr.WithTracing(c.Request().Context()).Errorf("got error during notificationService.NotifyRecordingChanged, %v", err)
			}
		}

		return nil
	}
}
