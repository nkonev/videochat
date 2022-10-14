package listener

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/montag451/go-eventbus"
	"github.com/streadway/amqp"
	"nkonev.name/chat/handlers/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/type_registry"
)

type NotificationsListener func(*amqp.Delivery) error

func CreateNotificationsListener(bus *eventbus.Bus, typeRegistry *type_registry.TypeRegistryInstance) NotificationsListener {
	return func(msg *amqp.Delivery) error {
		data := msg.Body
		s := string(data)
		typ := msg.Type
		Logger.Infof("Received %v with type %v", s, typ)

		if !typeRegistry.HasType(typ) {
			return errors.New(fmt.Sprintf("Unexpected type in tabbit fanout notifications: %v", typ))
		}

		instance := typeRegistry.MakeInstance(typ)

		switch instance.(type) {
		case dto.MessageNotify:
			bindTo := instance.(dto.MessageNotify)
			err := json.Unmarshal(data, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return nil
			}

			err = bus.PublishAsync(bindTo)
			if err != nil {
				Logger.Errorf("Error during sending to bus : %s", err)
			}
		}

		return nil
	}
}
