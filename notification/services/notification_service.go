package services

import (
	"nkonev.name/notification/db"
	"nkonev.name/notification/dto"
	. "nkonev.name/notification/logger"
)

type NotificationService struct {
	dbs db.DB
}

func CreateNotificationService(dbs db.DB) *NotificationService {
	return &NotificationService{
		dbs: dbs,
	}
}

func (srv *NotificationService) HandleChatNotification(event *dto.NotificationEvent) {
	if event.MentionNotification != nil {
		notification := event.MentionNotification
		notificationType := "mention"
		switch event.EventType {
		case "mention_added":
			err := srv.dbs.PutNotification(&notification.Id, event.UserId, event.ChatId, notificationType, &notification.Text)
			if err != nil {
				Logger.Errorf("Unable to put notification %v", err)
			}
		case "mention_deleted":
			err := srv.dbs.DeleteNotificationByMessageId(notification.Id, event.UserId)
			if err != nil {
				Logger.Errorf("Unable to delete notification %v", err)
			}
		}
	} else if event.MissedCallNotification {
		notificationType := "missed_call"
		err := srv.dbs.PutNotification(nil, event.UserId, event.ChatId, notificationType, nil)
		if err != nil {
			Logger.Errorf("Unable to put notification %v", err)
		}
	}

}
