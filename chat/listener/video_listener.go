package listener

import (
	"context"
	"encoding/json"
	"github.com/streadway/amqp"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
)

type VideoDialStatusListener func(*amqp.Delivery) error

func CreateVideoDialStatusListener(not services.Notifications, db db.DB) VideoDialStatusListener {
	return func(msg *amqp.Delivery) error {
		data := msg.Body
		s := string(data)
		Logger.Infof("Received %v", s)

		var bindTo = new(dto.VideoIsInvitingDto)
		err := json.Unmarshal(data, &bindTo)
		if err != nil {
			Logger.Errorf("Error during deserialize VideoIsInvitingDto %v", err)
			return nil
		}

		not.NotifyAboutDialStatus(context.Background(), bindTo.ChatId, bindTo.BehalfUserId, bindTo.Status, bindTo.UserIds)

		return nil
	}
}
