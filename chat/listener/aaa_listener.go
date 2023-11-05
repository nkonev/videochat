package listener

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
)

type AaaUserProfileUpdateListener func(*amqp.Delivery) error

func CreateAaaUserProfileUpdateListener(not services.Events) AaaUserProfileUpdateListener {
	return func(msg *amqp.Delivery) error {
		data := msg.Body
		s := string(data)
		Logger.Debugf("Received %v", s)

		var u *dto.UserAccountEvent
		err := json.Unmarshal(data, &u)
		if err != nil {
			Logger.Errorf("Error during deserialize UserAccountEvent %v", err)
			return nil
		}
		if u.EventType == "user_account_changed" {
			not.NotifyAboutProfileChanged(u.UserAccount)
		}

		return nil
	}
}
