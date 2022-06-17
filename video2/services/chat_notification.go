package services

import (
	"encoding/json"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
)

type NotificationService struct {
	rabbitMqPublisher *producer.RabbitPublisher
}

func NewNotificationService(producer *producer.RabbitPublisher) *NotificationService {
	return &NotificationService{
		rabbitMqPublisher: producer,
	}
}

// sent to chat through RabbitMQ
type chatNotifyDto struct {
	Data       *dto.NotifyDto `json:"data"`
	UsersCount int64          `json:"usersCount"`
	ChatId     int64          `json:"chatId"`
}

func (h *NotificationService) Notify(chatId, usersCount int64, data *dto.NotifyDto) error {
	var chatNotifyDto = chatNotifyDto{}
	if data != nil {
		Logger.Infof("Notifying with data chat_id=%v, login=%v, userId=%v", chatId, data.Login, data.UserId)
		chatNotifyDto.Data = data
	} else {
		Logger.Infof("Notifying without data chat_id=%v", chatId)
	}
	chatNotifyDto.UsersCount = usersCount
	chatNotifyDto.ChatId = chatId

	marshal, err := json.Marshal(chatNotifyDto)
	if err != nil {
		Logger.Error(err, "Failed during marshal chatNotifyDto")
		return err
	}

	return h.rabbitMqPublisher.Publish(marshal)
}
