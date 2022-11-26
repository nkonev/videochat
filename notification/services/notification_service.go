package services

import (
	"github.com/spf13/viper"
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
	err := srv.dbs.InitNotificationSettings(event.UserId)
	if err != nil {
		Logger.Errorf("Error during initializing notification settings %v", err)
		return
	}

	userNotificationsSettings, err := srv.dbs.GetNotificationSettings(event.UserId)
	if err != nil {
		Logger.Errorf("Error during initializing notification settings %v", err)
		return
	}

	if event.MentionNotification != nil && userNotificationsSettings.MentionsEnabled {
		notification := event.MentionNotification
		notificationType := "mention"
		switch event.EventType {
		case "mention_added":
			err := srv.removeExcessNotificationsIfNeed(event.UserId)
			if err != nil {
				Logger.Errorf("Unable to delete excess notifications %v", err)
				return
			}

			err = srv.dbs.PutNotification(&notification.Id, event.UserId, event.ChatId, notificationType, notification.Text)
			if err != nil {
				Logger.Errorf("Unable to put notification %v", err)
			}
		case "mention_deleted":
			err := srv.dbs.DeleteNotificationByMessageId(notification.Id, event.UserId)
			if err != nil {
				Logger.Errorf("Unable to delete notification %v", err)
			}
		}
	} else if event.MissedCallNotification != nil && userNotificationsSettings.MissedCallsEnabled {
		err := srv.removeExcessNotificationsIfNeed(event.UserId)
		if err != nil {
			Logger.Errorf("Unable to delete excess notifications %v", err)
			return
		}

		notification := event.MissedCallNotification
		notificationType := "missed_call"
		err = srv.dbs.PutNotification(nil, event.UserId, event.ChatId, notificationType, notification.Description)
		if err != nil {
			Logger.Errorf("Unable to put notification %v", err)
		}
	}

}

func (srv *NotificationService) removeExcessNotificationsIfNeed(userId int64) error {
	count, err := srv.dbs.GetNotificationCount(userId)
	if err != nil {
		Logger.Errorf("Unable to get notification count %v", err)
		return err
	}

	maxNotifications := viper.GetInt("maxNotifications")
	if count >= int64(maxNotifications) {
		toDelete := count - int64(maxNotifications) + 1
		return srv.dbs.DeleteExcessUserNotifications(userId, toDelete)
	}
	return nil
}
