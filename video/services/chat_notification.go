package services

import (
	"context"
	"nkonev.name/video/client"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
)

type NotificationService struct {
	rabbitMqUserCountPublisher *producer.RabbitUserCountPublisher
	rabbitMqRecordPublisher    *producer.RabbitRecordingPublisher
	restClient                 *client.RestClient
}

func NewNotificationService(producer *producer.RabbitUserCountPublisher, restClient *client.RestClient, rabbitMqRecordPublisher *producer.RabbitRecordingPublisher) *NotificationService {
	return &NotificationService{
		rabbitMqUserCountPublisher: producer,
		rabbitMqRecordPublisher:    rabbitMqRecordPublisher,
		restClient:                 restClient,
	}
}

func (h *NotificationService) NotifyVideoUserCountChanged(chatId, usersCount int64, ctx context.Context) error {
	Logger.Infof("Notifying video call chat_id=%v", chatId)

	var chatNotifyDto = dto.VideoCallUserCountChangedDto{
		UsersCount: usersCount,
		ChatId:     chatId,
	}

	participantIds, err := h.restClient.GetChatParticipantIds(chatId, ctx)
	if err != nil {
		Logger.Error(err, "Failed during getting chat participantIds")
		return err
	}

	return h.rabbitMqUserCountPublisher.Publish(participantIds, &chatNotifyDto, ctx)
}

func (h *NotificationService) NotifyRecordingChanged(chatId int64, recordInProgress bool, ctx context.Context) error {
	Logger.Infof("Notifying video call chat_id=%v", chatId)

	var chatNotifyDto = dto.VideoCallRecordingChangedDto{
		RecordInProgress: recordInProgress,
		ChatId:           chatId,
	}

	participantIds, err := h.restClient.GetChatParticipantIds(chatId, ctx)
	if err != nil {
		Logger.Error(err, "Failed during getting chat participantIds")
		return err
	}

	return h.rabbitMqRecordPublisher.Publish(participantIds, &chatNotifyDto, ctx)
}
