package listener

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/montag451/go-eventbus"
	"github.com/streadway/amqp"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/type_registry"
)

type NotificationsListener func(*amqp.Delivery) error

func CreateNotificationsListener(bus *eventbus.Bus, typeRegistry *type_registry.TypeRegistryInstance) NotificationsListener {
	return func(msg *amqp.Delivery) error {
		bytesData := msg.Body
		strData := string(bytesData)
		aType := msg.Type
		Logger.Infof("Received %v with type %v", strData, aType)

		if !typeRegistry.HasType(aType) {
			return errors.New(fmt.Sprintf("Unexpected type in tabbit fanout notifications: %v", aType))
		}

		anInstance := typeRegistry.MakeInstance(aType)

		switch bindTo := anInstance.(type) {
		case dto.MessageNotify:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return err
			}

			err = bus.PublishAsync(bindTo)
			if err != nil {
				Logger.Errorf("Error during sending to bus : %v", err)
				return err
			}
		default:
			Logger.Errorf("Unexpected type : %v", anInstance)
		}

		return nil
	}
}
