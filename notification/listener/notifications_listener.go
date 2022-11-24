package listener

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"nkonev.name/notification/db"
	"nkonev.name/notification/dto"
	. "nkonev.name/notification/logger"
)

type NotificationsListener func(*amqp.Delivery) error

func CreateNotificationsListener(db db.DB) NotificationsListener {
	return func(msg *amqp.Delivery) error {
		bytesData := msg.Body
		strData := string(bytesData)
		aType := msg.Type
		Logger.Infof("Received %v with type %v", strData, aType)

		var bindTo = new(dto.NotificationEvent)
		err := json.Unmarshal(msg.Body, bindTo)
		if err != nil {
			Logger.Errorf("Unable to unmarshall notification %v", err)
			return err
		}

		if bindTo.MentionNotification != nil {
			notification := bindTo.MentionNotification
			notificationType := "mention"
			switch bindTo.EventType {
			case "mention_added":
				err := db.PutNotification(&notification.Id, bindTo.UserId, bindTo.ChatId, notificationType, notification.Text)
				if err != nil {
					Logger.Errorf("Unable to put notification %v", err)
					return err
				}
			case "mention_deleted":
				err := db.DeleteNotificationByMessageId(notification.Id, bindTo.UserId)
				if err != nil {
					Logger.Errorf("Unable to delete notification %v", err)
					return err
				}
			}
		}

		return nil
	}
}
