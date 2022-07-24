package listener

import (
	"encoding/json"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/notifications"
)

type VideoListener func(data []byte) error

func CreateVideoListener(not notifications.Notifications, db db.DB) VideoListener {
	return func(data []byte) error {
		s := string(data)
		Logger.Infof("Received %v", s)

		var bindTo = new(dto.ChatNotifyDto)
		err := json.Unmarshal(data, &bindTo)
		if err != nil {
			Logger.Errorf("Error during deserialize ChatNotifyDto %v", err)
			return nil
		}
		ids, err := db.GetAllParticipantIds(bindTo.ChatId)
		if err != nil {
			Logger.Warnf("Error during get participants of chat %v", bindTo.ChatId)
			return err
		}

		not.NotifyAboutVideoCallChanged(*bindTo, ids)

		return nil
	}
}
