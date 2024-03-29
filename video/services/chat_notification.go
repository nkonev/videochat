package services

import (
	"context"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
)

type NotificationService struct {
	rabbitMqUserCountPublisher *producer.RabbitUserCountPublisher
	rabbitMqRecordPublisher    *producer.RabbitRecordingPublisher
	rabbitMqScreenSharePublisher *producer.RabbitScreenSharePublisher
	rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher
}

func NewNotificationService(
	producer *producer.RabbitUserCountPublisher,
	rabbitMqRecordPublisher *producer.RabbitRecordingPublisher,
	rabbitMqScreenSharePublisher *producer.RabbitScreenSharePublisher,
	rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher,
) *NotificationService {
	return &NotificationService{
		rabbitMqUserCountPublisher: producer,
		rabbitMqScreenSharePublisher: rabbitMqScreenSharePublisher,
		rabbitMqRecordPublisher:    rabbitMqRecordPublisher,
		rabbitUserIdsPublisher: rabbitUserIdsPublisher,
	}
}

// sends notification about video users, which is showed
// as a small handset in ChatList
// and as a number of video users in badge near call button
func (h *NotificationService) NotifyVideoUserCountChanged(participantIds []int64, chatId, usersCount int64, ctx context.Context) error {
	Logger.Debugf("Notifying video call chat_id=%v", chatId)

	var chatNotifyDto = dto.VideoCallUserCountChangedDto{
		UsersCount: usersCount,
		ChatId:     chatId,
	}

	return h.rabbitMqUserCountPublisher.Publish(participantIds, &chatNotifyDto, ctx)
}

func (h *NotificationService) NotifyVideoScreenShareChanged(participantIds []int64, chatId int64, hasScreenShares bool, ctx context.Context) error {
	Logger.Debugf("Notifying video call chat_id=%v", chatId)

	var chatNotifyDto = dto.VideoCallScreenShareChangedDto{
		HasScreenShares: hasScreenShares,
		ChatId:     chatId,
	}

	return h.rabbitMqScreenSharePublisher.Publish(participantIds, &chatNotifyDto, ctx)
}


func (h *NotificationService) NotifyRecordingChanged(chatId int64, recordInProgressByOwner map[int64]bool, ctx context.Context) error {
	Logger.Debugf("Notifying video call chat_id=%v", chatId)

	return h.rabbitMqRecordPublisher.Publish(recordInProgressByOwner, chatId, ctx)
}
