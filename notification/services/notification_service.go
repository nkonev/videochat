package services

import (
	"context"
	"database/sql"
	"errors"
	"github.com/spf13/viper"
	"nkonev.name/notification/db"
	"nkonev.name/notification/dto"
	. "nkonev.name/notification/logger"
	"nkonev.name/notification/producer"
)

type NotificationService struct {
	dbs                   *db.DB
	rabbitEventsPublisher *producer.RabbitEventPublisher
}

func CreateNotificationService(dbs *db.DB, rabbitEventsPublisher *producer.RabbitEventPublisher) *NotificationService {
	return &NotificationService{
		dbs:                   dbs,
		rabbitEventsPublisher: rabbitEventsPublisher,
	}
}

const NotificationAdd = "notification_add"
const NotificationDelete = "notification_delete"

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
		mentionNotification := event.MentionNotification
		notificationType := "mention"
		switch event.EventType {
		case "mention_added":
			err := srv.removeExcessNotificationsIfNeed(event.UserId)
			if err != nil {
				Logger.Errorf("Unable to delete excess notifications %v", err)
				return
			}

			id, createDateTime, err := srv.dbs.PutNotification(&mentionNotification.Id, event.UserId, event.ChatId, notificationType, mentionNotification.Text, event.ByUserId, event.ByLogin, event.ChatTitle)
			if err != nil {
				Logger.Errorf("Unable to put notification %v", err)
				return
			}
			err = srv.rabbitEventsPublisher.Publish(event.UserId, &dto.NotificationDto{
				Id:               id,
				ChatId:           event.ChatId,
				MessageId:        &mentionNotification.Id,
				NotificationType: notificationType,
				Description:      mentionNotification.Text,
				CreateDateTime:   createDateTime,
				ByUserId:         event.ByUserId,
				ByLogin:          event.ByLogin,
				ChatTitle:        event.ChatTitle,
			}, NotificationAdd, context.Background())
			if err != nil {
				Logger.Errorf("Unable to send notification delete %v", err)
			}

		case "mention_deleted":
			id, err := srv.dbs.DeleteNotificationByMessageId(mentionNotification.Id, notificationType, event.UserId)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) { // occurs during message read on previously read message
					Logger.Debugf("Missed notification %v", err)
				} else {
					Logger.Errorf("Unable to delete notification %v", err)
				}
				return
			}

			err = srv.rabbitEventsPublisher.Publish(event.UserId, dto.NewNotificationDeleteDto(id), NotificationDelete, context.Background())
			if err != nil {
				Logger.Errorf("Unable to send notification delete %v", err)
			}
		default:
			Logger.Errorf("Unexpected event type %v", event.EventType)
		}
	} else if event.MissedCallNotification != nil && userNotificationsSettings.MissedCallsEnabled {
		err := srv.removeExcessNotificationsIfNeed(event.UserId)
		if err != nil {
			Logger.Errorf("Unable to delete excess notifications %v", err)
			return
		}

		notification := event.MissedCallNotification
		notificationType := "missed_call"
		id, createDateTime, err := srv.dbs.PutNotification(nil, event.UserId, event.ChatId, notificationType, notification.Description, event.ByUserId, event.ByLogin, event.ChatTitle)
		if err != nil {
			Logger.Errorf("Unable to put notification %v", err)
			return
		}

		err = srv.rabbitEventsPublisher.Publish(event.UserId, &dto.NotificationDto{
			Id:               id,
			ChatId:           event.ChatId,
			MessageId:        nil,
			NotificationType: notificationType,
			Description:      notification.Description,
			CreateDateTime:   createDateTime,
			ByUserId:         event.ByUserId,
			ByLogin:          event.ByLogin,
			ChatTitle:        event.ChatTitle,
		}, NotificationAdd, context.Background())
		if err != nil {
			Logger.Errorf("Unable to send notification delete %v", err)
		}
	} else if event.ReplyNotification != nil && userNotificationsSettings.AnswersEnabled {
		err := srv.removeExcessNotificationsIfNeed(event.UserId)
		if err != nil {
			Logger.Errorf("Unable to delete excess notifications %v", err)
			return
		}

		notification := event.ReplyNotification
		notificationType := "reply"
		switch event.EventType {
		case "reply_added":
			id, createDateTime, err := srv.dbs.PutNotification(&notification.MessageId, event.UserId, event.ChatId, notificationType, notification.ReplyableMessage, event.ByUserId, event.ByLogin, event.ChatTitle)
			if err != nil {
				Logger.Errorf("Unable to put notification %v", err)
				return
			}
			err = srv.rabbitEventsPublisher.Publish(event.UserId, &dto.NotificationDto{
				Id:               id,
				ChatId:           event.ChatId,
				MessageId:        &notification.MessageId,
				NotificationType: notificationType,
				Description:      notification.ReplyableMessage,
				CreateDateTime:   createDateTime,
				ByUserId:         event.ByUserId,
				ByLogin:          event.ByLogin,
				ChatTitle:        event.ChatTitle,
			}, NotificationAdd, context.Background())
			if err != nil {
				Logger.Errorf("Unable to send notification delete %v", err)
			}

		case "reply_deleted":
			id, err := srv.dbs.DeleteNotificationByMessageId(notification.MessageId, notificationType, event.UserId)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) { // occurs during message read on previously read message
					Logger.Debugf("Missed notification %v", err)
				} else {
					Logger.Errorf("Unable to delete notification %v", err)
				}
				return
			}
			err = srv.rabbitEventsPublisher.Publish(event.UserId, dto.NewNotificationDeleteDto(id), NotificationDelete, context.Background())
			if err != nil {
				Logger.Errorf("Unable to send notification delete %v", err)
			}
		default:
			Logger.Errorf("Unexpected event type %v", event.EventType)
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
		notificationsIdsToDelete, err := srv.dbs.GetExcessUserNotificationIds(userId, toDelete)
		if err != nil {
			Logger.Errorf("Unable to get notification ids to delete %v", err)
			return err
		}
		for _, id := range notificationsIdsToDelete {
			err := srv.dbs.DeleteNotification(id, userId)
			if err != nil {
				Logger.Errorf("Unable to delete notification %v", err)
				return err
			}
			err = srv.rabbitEventsPublisher.Publish(userId, dto.NewNotificationDeleteDto(id), NotificationDelete, context.Background())
			if err != nil {
				Logger.Errorf("Unable to send notification delete %v", err)
				return err
			}
		}
	}
	return nil
}
