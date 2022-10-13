package listener

import (
	"github.com/montag451/go-eventbus"
	"github.com/streadway/amqp"
	"nkonev.name/chat/db"
)

type NotificationsListener func(*amqp.Delivery) error

func CreateNotificationsListener(bus *eventbus.Bus, db db.DB) NotificationsListener {
	return func(msg *amqp.Delivery) error {
		/*data := msg.Body
		s := string(data)
		typ := msg.Type
		Logger.Infof("Received %v with type %v", s, typ)

		intType := reflect.TypeOf(typ)
		bindTo := reflect.New(intType)

		//var bindTo = new(services.MessageNotify)
		err := json.Unmarshal(data, &bindTo)
		if err != nil {
			Logger.Errorf("Error during deserialize ChatNotifyDto %v", err)
			return nil
		}

		err := bus.PublishAsync(services.MessageNotify{
			Type:                eventType,
			MessageNotification: message,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to bus : %s", err)
		}*/

		return nil
	}
}
