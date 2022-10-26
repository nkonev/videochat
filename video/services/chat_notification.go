package services

import (
	"context"
	"nkonev.name/video/client"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
)

type NotificationService struct {
	rabbitMqPublisher *producer.RabbitNotificationsPublisher
	restClient        *client.RestClient
}

func NewNotificationService(producer *producer.RabbitNotificationsPublisher, restClient *client.RestClient) *NotificationService {
	return &NotificationService{
		rabbitMqPublisher: producer,
		restClient:        restClient,
	}
}

func (h *NotificationService) Notify(chatId, usersCount int64, ctx context.Context) error {
	var chatNotifyDto = dto.VideoCallChangedDto{}
	Logger.Infof("Notifying without data chat_id=%v", chatId)

	chatNotifyDto.UsersCount = usersCount
	chatNotifyDto.ChatId = chatId

	participantIds, err := h.restClient.GetChatParticipantIds(chatId, ctx)
	if err != nil {
		Logger.Error(err, "Failed during getting chat participantIds")
		return err
	}

	return h.rabbitMqPublisher.Publish(participantIds, &chatNotifyDto, ctx)
}
