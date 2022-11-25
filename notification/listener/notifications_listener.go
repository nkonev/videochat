package listener

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"nkonev.name/notification/dto"
	. "nkonev.name/notification/logger"
	"nkonev.name/notification/services"
)

type NotificationsListener func(*amqp.Delivery) error

func CreateNotificationsListener(service *services.NotificationService) NotificationsListener {
	return func(msg *amqp.Delivery) error {
		bytesData := msg.Body
		strData := string(bytesData)
		Logger.Debugf("Received %v", strData)

		var bindTo = new(dto.NotificationEvent)
		err := json.Unmarshal(msg.Body, bindTo)
		if err != nil {
			Logger.Errorf("Unable to unmarshall notification %v", err)
			return err
		}

		service.HandleChatNotification(bindTo)

		return nil
	}
}
