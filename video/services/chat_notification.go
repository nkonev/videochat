package services

import (
	"context"
	log "github.com/sirupsen/logrus"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
)

type NotificationService struct {
	rabbitMqUserCountPublisher   *producer.RabbitUserCountPublisher
	rabbitMqRecordPublisher      *producer.RabbitRecordingPublisher
	rabbitMqScreenSharePublisher *producer.RabbitScreenSharePublisher
	rabbitUserIdsPublisher       *producer.RabbitUserIdsPublisher
	lgr                          *log.Logger
}

func NewNotificationService(
	producer *producer.RabbitUserCountPublisher,
	rabbitMqRecordPublisher *producer.RabbitRecordingPublisher,
	rabbitMqScreenSharePublisher *producer.RabbitScreenSharePublisher,
	rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher,
	lgr *log.Logger,
) *NotificationService {
	return &NotificationService{
		rabbitMqUserCountPublisher:   producer,
		rabbitMqScreenSharePublisher: rabbitMqScreenSharePublisher,
		rabbitMqRecordPublisher:      rabbitMqRecordPublisher,
		rabbitUserIdsPublisher:       rabbitUserIdsPublisher,
		lgr:                          lgr,
	}
}

// sends notification about video users, which is showed
// as a small handset in ChatList
// and as a number of video users in badge near call button
func (h *NotificationService) NotifyVideoUserCountChanged(ctx context.Context, participantIds []int64, chatId, usersCount int64) error {
	GetLogEntry(ctx, h.lgr).Debugf("Notifying video call chat_id=%v", chatId)

	var chatNotifyDto = dto.VideoCallUserCountChangedDto{
		UsersCount: usersCount,
		ChatId:     chatId,
	}

	return h.rabbitMqUserCountPublisher.Publish(ctx, participantIds, &chatNotifyDto)
}

func (h *NotificationService) NotifyVideoScreenShareChanged(ctx context.Context, participantIds []int64, chatId int64, hasScreenShares bool) error {
	GetLogEntry(ctx, h.lgr).Debugf("Notifying video call chat_id=%v", chatId)

	var chatNotifyDto = dto.VideoCallScreenShareChangedDto{
		HasScreenShares: hasScreenShares,
		ChatId:          chatId,
	}

	return h.rabbitMqScreenSharePublisher.Publish(ctx, participantIds, &chatNotifyDto)
}

func (h *NotificationService) NotifyRecordingChanged(ctx context.Context, chatId int64, recordInProgressByOwner map[int64]bool) error {
	GetLogEntry(ctx, h.lgr).Debugf("Notifying video call chat_id=%v", chatId)

	return h.rabbitMqRecordPublisher.Publish(ctx, recordInProgressByOwner, chatId)
}
