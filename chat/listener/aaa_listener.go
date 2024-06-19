package listener

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
	"nkonev.name/chat/type_registry"
)

type AaaUserProfileUpdateListener func(*amqp.Delivery) error

func CreateAaaUserProfileUpdateListener(not *services.Events, typeRegistry *type_registry.TypeRegistryInstance, db *db.DB) AaaUserProfileUpdateListener {
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
		case dto.UserAccountEventGroup:
			err := json.Unmarshal(bytesData, &bindTo)
			if err != nil {
				Logger.Errorf("Error during deserialize notification %v", err)
				return err
			}
			if bindTo.EventType == "user_account_changed" {
				not.NotifyAboutProfileChanged(bindTo.ForRoleUser, db)
			}

		default:
			Logger.Errorf("Unexpected type : %v", anInstance)
			return errors.New(fmt.Sprintf("Unexpected type : %v", anInstance))

		}

		return nil
	}
}
