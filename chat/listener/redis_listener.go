package listener

import (
	"encoding/json"
	"nkonev.name/chat/handlers/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/notifications"
)

type AaaUserProfileUpdateListener func(channel string, data []byte) error

func CreateAaaUserProfileUpdateListener(not notifications.Notifications) AaaUserProfileUpdateListener {
	return func(channel string, data []byte) error {
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
