package services

import (
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
)

type NotificationService struct {
	rabbitMqPublisher *producer.RabbitNotificationsPublisher
}

func NewNotificationService(producer *producer.RabbitNotificationsPublisher) *NotificationService {
	return &NotificationService{
		rabbitMqPublisher: producer,
	}
}

func (h *NotificationService) Notify(chatId, usersCount int64, data *dto.NotifyDto) error {
	var chatNotifyDto = dto.ChatNotifyDto{}
	if data != nil {
		Logger.Infof("Notifying with data chat_id=%v, login=%v, userId=%v", chatId, data.Login, data.UserId)
		chatNotifyDto.Data = data
	} else {
		Logger.Infof("Notifying without data chat_id=%v", chatId)
	}
	chatNotifyDto.UsersCount = usersCount
	chatNotifyDto.ChatId = chatId

	return h.rabbitMqPublisher.Publish(&chatNotifyDto)
}
