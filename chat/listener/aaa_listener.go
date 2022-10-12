package listener

import (
	"encoding/json"
	"nkonev.name/chat/handlers/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
)

type AaaUserProfileUpdateListener func(data []byte) error

func CreateAaaUserProfileUpdateListener(not services.Notifications) AaaUserProfileUpdateListener {
	return func(data []byte) error {
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
