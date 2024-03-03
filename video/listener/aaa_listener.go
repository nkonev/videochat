package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
	"nkonev.name/video/type_registry"
)

type AaaUserProfileUpdateListener func(*amqp.Delivery) error

func CreateAaaUserProfileUpdateListener(userService *services.UserService, typeRegistry *type_registry.TypeRegistryInstance) AaaUserProfileUpdateListener {
	return func(msg *amqp.Delivery) error {
		bytesData := msg.Body
		strData := string(bytesData)
		aType := msg.Type
		Logger.Debugf("Received %v with type %v", strData, aType)

		if !typeRegistry.HasType(aType) {
			errStr := fmt.Sprintf("Unexpected type in rabbit fanout notifications: %v", aType)
			Logger.Debugf(errStr)
			return nil
		}

		anInstance := typeRegistry.MakeInstance(aType)

		switch bindTo := anInstance.(type) {
		case dto.UserSessionsKilledEvent:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return err
			}
			if bindTo.EventType == "user_sessions_killed" {
				userService.KickUser(context.Background(), bindTo.UserId)
			}

		default:
			Logger.Errorf("Unexpected type : %v", anInstance)
			return errors.New(fmt.Sprintf("Unexpected type : %v", anInstance))

		}

		return nil
	}
}
