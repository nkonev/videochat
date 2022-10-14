package listener

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
)

type AaaUserProfileUpdateListener func(*amqp.Delivery) error

func CreateAaaUserProfileUpdateListener(not services.Notifications) AaaUserProfileUpdateListener {
	return func(msg *amqp.Delivery) error {
		data := msg.Body
		s := string(data)
		Logger.Infof("Received %v", s)

		var u *dto.User
		err := json.Unmarshal(data, &u)
		if err != nil {
			Logger.Errorf("Error during deserialize User %v", err)
			return nil
		}
		not.NotifyAboutProfileChanged(u)

		return nil
	}
}
