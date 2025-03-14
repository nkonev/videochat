package services

import (
	"context"
	"nkonev.name/notification/db"
	"nkonev.name/notification/dto"
	"nkonev.name/notification/logger"
	"nkonev.name/notification/producer"
)

type NotificationEphemeralService struct {
	dbs                   *db.DB
	rabbitInvitePublisher *producer.RabbitInvitePublisher
	lgr                   *logger.Logger
}

func CreateNotificationEphemeralService(dbs *db.DB, rabbitInvitePublisher *producer.RabbitInvitePublisher, lgr *logger.Logger) *NotificationEphemeralService {
	return &NotificationEphemeralService{
		dbs:                   dbs,
		rabbitInvitePublisher: rabbitInvitePublisher,
		lgr:                   lgr,
	}
}

func (s *NotificationEphemeralService) HandleChatNotification(ctx context.Context, event *dto.NotificationEphemeralEvent) {
	if event.VideoChatInvitation != nil {
		err := s.rabbitInvitePublisher.Publish(ctx, event.EventType, event.VideoChatInvitation, event.UserId)
		if err != nil {
			s.lgr.WithTracing(ctx).Error(err, "Error during publishing")
		}
	}
}
