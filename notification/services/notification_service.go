package services

import (
	"context"
	"database/sql"
	"errors"
	"github.com/spf13/viper"
	"nkonev.name/notification/db"
	"nkonev.name/notification/dto"
	"nkonev.name/notification/logger"
	"nkonev.name/notification/producer"
)

type NotificationService struct {
	dbs                   *db.DB
	rabbitEventsPublisher *producer.RabbitEventPublisher
	lgr                   *logger.Logger
}

func CreateNotificationService(dbs *db.DB, rabbitEventsPublisher *producer.RabbitEventPublisher, lgr *logger.Logger) *NotificationService {
	return &NotificationService{
		dbs:                   dbs,
		rabbitEventsPublisher: rabbitEventsPublisher,
		lgr:                   lgr,
	}
}

const NotificationAdd = "notification_add"
const NotificationDelete = "notification_delete"
const NotificationClearAll = "notification_clear_all"

func (srv *NotificationService) HandleChatNotification(ctx context.Context, event *dto.NotificationEvent) {

	settings, err := srv.getNotificationSettings(ctx, event)
	if err != nil {
		srv.lgr.WithTracing(ctx).Errorf("Unable to get notification settings %v", err)
		return
	}

	var count int64

	if event.MentionNotification != nil && settings.MentionsEnabled {
		mentionNotification := event.MentionNotification
		notificationType := "mention"
		switch event.EventType {
		case "mention_added":
			err := srv.removeExcessNotificationsIfNeed(ctx, event.UserId)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to delete excess notifications %v", err)
				return
			}

			id, createDateTime, err := srv.dbs.PutNotification(ctx, &mentionNotification.Id, event.UserId, event.ChatId, notificationType, mentionNotification.Text, event.ByUserId, event.ByLogin, event.ChatTitle, nil)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to put notification %v", err)
				return
			}

			count, err = srv.dbs.GetNotificationCount(ctx, event.UserId)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to count notification %v", err)
				return
			}

			err = srv.rabbitEventsPublisher.Publish(
				ctx,
				event.UserId,
				&dto.WrapperNotificationDto{
					NotificationDto: dto.NotificationDto{
						Id:               id,
						ChatId:           event.ChatId,
						MessageId:        &mentionNotification.Id,
						NotificationType: notificationType,
						Description:      mentionNotification.Text,
						CreateDateTime:   createDateTime,
						ByUserId:         event.ByUserId,
						ByLogin:          event.ByLogin,
						ByAvatar:         event.ByAvatar,
						ChatTitle:        event.ChatTitle,
					},
					TotalCount: count,
				},
				NotificationAdd,
			)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to send notification delete %v", err)
			}

		case "mention_deleted":
			id, err := srv.dbs.DeleteNotificationByMessageId(ctx, mentionNotification.Id, notificationType, event.UserId, nil)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) { // occurs during message read on previously read message
					srv.lgr.WithTracing(ctx).Debugf("Missed notification %v", err)
				} else {
					srv.lgr.WithTracing(ctx).Errorf("Unable to delete notification %v", err)
				}
				return
			}

			count, err = srv.dbs.GetNotificationCount(ctx, event.UserId)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to count notification %v", err)
				return
			}

			err = srv.rabbitEventsPublisher.Publish(ctx, event.UserId, dto.NewWrapperNotificationDeleteDto(id, count, notificationType), NotificationDelete)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to send notification delete %v", err)
			}
		default:
			srv.lgr.WithTracing(ctx).Errorf("Unexpected event type %v", event.EventType)
		}
	} else if event.MissedCallNotification != nil && settings.MissedCallsEnabled {
		err := srv.removeExcessNotificationsIfNeed(ctx, event.UserId)
		if err != nil {
			srv.lgr.WithTracing(ctx).Errorf("Unable to delete excess notifications %v", err)
			return
		}

		notification := event.MissedCallNotification
		notificationType := "missed_call"
		id, createDateTime, err := srv.dbs.PutNotification(ctx, nil, event.UserId, event.ChatId, notificationType, notification.Description, event.ByUserId, event.ByLogin, event.ChatTitle, nil)
		if err != nil {
			srv.lgr.WithTracing(ctx).Errorf("Unable to put notification %v", err)
			return
		}

		count, err = srv.dbs.GetNotificationCount(ctx, event.UserId)
		if err != nil {
			srv.lgr.WithTracing(ctx).Errorf("Unable to count notification %v", err)
			return
		}

		err = srv.rabbitEventsPublisher.Publish(
			ctx,
			event.UserId,
			&dto.WrapperNotificationDto{
				NotificationDto: dto.NotificationDto{
					Id:               id,
					ChatId:           event.ChatId,
					MessageId:        nil,
					NotificationType: notificationType,
					Description:      notification.Description,
					CreateDateTime:   createDateTime,
					ByUserId:         event.ByUserId,
					ByLogin:          event.ByLogin,
					ByAvatar:         event.ByAvatar,
					ChatTitle:        event.ChatTitle,
				},
				TotalCount: count,
			},
			NotificationAdd,
		)
		if err != nil {
			srv.lgr.WithTracing(ctx).Errorf("Unable to send notification delete %v", err)
		}
	} else if event.ReplyNotification != nil && settings.AnswersEnabled {
		err := srv.removeExcessNotificationsIfNeed(ctx, event.UserId)
		if err != nil {
			srv.lgr.WithTracing(ctx).Errorf("Unable to delete excess notifications %v", err)
			return
		}

		notification := event.ReplyNotification
		notificationType := "reply"
		switch event.EventType {
		case "reply_added":
			id, createDateTime, err := srv.dbs.PutNotification(ctx, &notification.MessageId, event.UserId, event.ChatId, notificationType, notification.ReplyableMessage, event.ByUserId, event.ByLogin, event.ChatTitle, nil)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to put notification %v", err)
				return
			}
			count, err = srv.dbs.GetNotificationCount(ctx, event.UserId)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to count notification %v", err)
				return
			}

			err = srv.rabbitEventsPublisher.Publish(
				ctx,
				event.UserId,
				&dto.WrapperNotificationDto{
					NotificationDto: dto.NotificationDto{
						Id:               id,
						ChatId:           event.ChatId,
						MessageId:        &notification.MessageId,
						NotificationType: notificationType,
						Description:      notification.ReplyableMessage,
						CreateDateTime:   createDateTime,
						ByUserId:         event.ByUserId,
						ByLogin:          event.ByLogin,
						ByAvatar:         event.ByAvatar,
						ChatTitle:        event.ChatTitle,
					},
					TotalCount: count,
				},
				NotificationAdd,
			)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to send notification delete %v", err)
			}

		case "reply_deleted":
			id, err := srv.dbs.DeleteNotificationByMessageId(ctx, notification.MessageId, notificationType, event.UserId, nil)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) { // occurs during message read on previously read message
					srv.lgr.WithTracing(ctx).Debugf("Missed notification %v", err)
				} else {
					srv.lgr.WithTracing(ctx).Errorf("Unable to delete notification %v", err)
				}
				return
			}
			count, err = srv.dbs.GetNotificationCount(ctx, event.UserId)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to count notification %v", err)
				return
			}

			err = srv.rabbitEventsPublisher.Publish(ctx, event.UserId, dto.NewWrapperNotificationDeleteDto(id, count, notificationType), NotificationDelete)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to send notification delete %v", err)
			}
		default:
			srv.lgr.WithTracing(ctx).Errorf("Unexpected event type %v", event.EventType)
		}

	} else if event.ReactionEvent != nil && settings.ReactionsEnabled {
		err := srv.removeExcessNotificationsIfNeed(ctx, event.UserId)
		if err != nil {
			srv.lgr.WithTracing(ctx).Errorf("Unable to delete excess notifications %v", err)
			return
		}
		notification := event.ReactionEvent
		notificationType := "reaction"

		switch event.EventType {
		case "reaction_notification_added":
			id, createDateTime, err := srv.dbs.PutNotification(ctx, &notification.MessageId, event.UserId, event.ChatId, notificationType, notification.Reaction, event.ByUserId, event.ByLogin, event.ChatTitle, &notification.Reaction)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to put notification %v", err)
				return
			}
			count, err = srv.dbs.GetNotificationCount(ctx, event.UserId)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to count notification %v", err)
				return
			}

			err = srv.rabbitEventsPublisher.Publish(
				ctx,
				event.UserId,
				&dto.WrapperNotificationDto{
					NotificationDto: dto.NotificationDto{
						Id:               id,
						ChatId:           event.ChatId,
						MessageId:        &notification.MessageId,
						NotificationType: notificationType,
						Description:      notification.Reaction,
						CreateDateTime:   createDateTime,
						ByUserId:         event.ByUserId,
						ByLogin:          event.ByLogin,
						ByAvatar:         event.ByAvatar,
						ChatTitle:        event.ChatTitle,
					},
					TotalCount: count,
				},
				NotificationAdd,
			)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to send notification delete %v", err)
			}

		case "reaction_notification_removed":
			id, err := srv.dbs.DeleteNotificationByMessageId(ctx, notification.MessageId, notificationType, event.UserId, &notification.Reaction)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) { // occurs during message read on previously read message
					srv.lgr.WithTracing(ctx).Debugf("Missed notification %v", err)
				} else {
					srv.lgr.WithTracing(ctx).Errorf("Unable to delete notification %v", err)
				}
				return
			}
			count, err = srv.dbs.GetNotificationCount(ctx, event.UserId)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to count notification %v", err)
				return
			}

			err = srv.rabbitEventsPublisher.Publish(ctx, event.UserId, dto.NewWrapperNotificationDeleteDto(id, count, notificationType), NotificationDelete)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to send notification delete %v", err)
			}
		}
	}

}

func (srv *NotificationService) getNotificationSettings(ctx context.Context, event *dto.NotificationEvent) (*dto.NotificationGlobalSettings, error) {

	userNotificationsGlobalSettings, err := srv.dbs.GetNotificationGlobalSettings(ctx, event.UserId)
	if err != nil {
		srv.lgr.WithTracing(ctx).Errorf("Error during getting global notification settings %v", err)
		return nil, err
	}

	userNotificationsPerChatSettings, err := srv.dbs.GetNotificationPerChatSettings(ctx, event.UserId, event.ChatId)
	if err != nil {
		srv.lgr.WithTracing(ctx).Errorf("Error during getting per chat notification settings %v", err)
		return nil, err
	}

	result := dto.NotificationGlobalSettings{
		MentionsEnabled:    userNotificationsGlobalSettings.MentionsEnabled,
		MissedCallsEnabled: userNotificationsGlobalSettings.MissedCallsEnabled,
		AnswersEnabled:     userNotificationsGlobalSettings.AnswersEnabled,
		ReactionsEnabled:   userNotificationsGlobalSettings.ReactionsEnabled,
	}

	// override
	if userNotificationsPerChatSettings.MentionsEnabled != nil {
		result.MentionsEnabled = *userNotificationsPerChatSettings.MentionsEnabled
	}

	if userNotificationsPerChatSettings.MissedCallsEnabled != nil {
		result.MissedCallsEnabled = *userNotificationsPerChatSettings.MissedCallsEnabled
	}

	if userNotificationsPerChatSettings.AnswersEnabled != nil {
		result.AnswersEnabled = *userNotificationsPerChatSettings.AnswersEnabled
	}

	if userNotificationsPerChatSettings.ReactionsEnabled != nil {
		result.ReactionsEnabled = *userNotificationsPerChatSettings.ReactionsEnabled
	}

	return &result, nil
}

func (srv *NotificationService) removeExcessNotificationsIfNeed(ctx context.Context, userId int64) error {
	count, err := srv.dbs.GetNotificationCount(ctx, userId)
	if err != nil {
		srv.lgr.WithTracing(ctx).Errorf("Unable to get notification count %v", err)
		return err
	}

	maxNotifications := viper.GetInt("maxNotificationsPerUser")
	if count >= int64(maxNotifications) {
		srv.lgr.WithTracing(ctx).Infof("Notifications %v are exceeded maxNotificationsPerUser %v, going to delete the oldest", count, maxNotifications)

		toDelete := count - int64(maxNotifications) + 1
		notificationsIdsToDelete, err := srv.dbs.GetExcessUserNotificationIds(ctx, userId, toDelete)
		if err != nil {
			srv.lgr.WithTracing(ctx).Errorf("Unable to get notification ids to delete %v", err)
			return err
		}
		for _, id := range notificationsIdsToDelete {
			deletedNotificationType, err := srv.dbs.DeleteNotification(ctx, id, userId)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to delete notification %v", err)
				return err
			}
			count, err = srv.dbs.GetNotificationCount(ctx, userId)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to count notification %v", err)
				return err
			}

			err = srv.rabbitEventsPublisher.Publish(ctx, userId, dto.NewWrapperNotificationDeleteDto(id, count, deletedNotificationType), NotificationDelete)
			if err != nil {
				srv.lgr.WithTracing(ctx).Errorf("Unable to send notification delete %v", err)
				return err
			}
		}
	}
	return nil
}
