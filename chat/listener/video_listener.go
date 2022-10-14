package listener

import (
	"context"
	"encoding/json"
	"github.com/streadway/amqp"
	"nkonev.name/chat/db"
	dto2 "nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
	"nkonev.name/chat/utils"
)

type VideoNotificationsListener func(*amqp.Delivery) error

type VideoInviteListener func(*amqp.Delivery) error

type VideoDialStatusListener func(*amqp.Delivery) error

func CreateVideoCallChangedListener(not services.Notifications, db db.DB) VideoNotificationsListener {
	return func(msg *amqp.Delivery) error {
		data := msg.Body
		s := string(data)
		Logger.Infof("Received %v", s)

		var bindTo = new(dto2.ChatNotifyDto)
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

type simpleChat struct {
	Id        int64  `json:"id"`
	Name      string `json:"name"`
	IsTetATet bool   `json:"tetATet"`
}

func (r *simpleChat) GetId() int64 {
	return r.Id
}

func (r *simpleChat) GetName() string {
	return r.Name
}

func (r *simpleChat) SetName(s string) {
	r.Name = s
}

func (r *simpleChat) GetIsTetATet() bool {
	return r.IsTetATet
}

func CreateVideoInviteListener(not services.Notifications, db db.DB) VideoInviteListener {
	return func(msg *amqp.Delivery) error {
		data := msg.Body
		s := string(data)
		Logger.Infof("Received %v", s)

		var bindTo = new(dto2.VideoInviteDto)
		err := json.Unmarshal(data, &bindTo)
		if err != nil {
			Logger.Errorf("Error during deserialize VideoInviteDto %v", err)
			return nil
		}

		chat, err := db.GetChat(bindTo.BehalfUserId, bindTo.ChatId)
		if err != nil {
			return err
		}

		meAsUser := dto2.User{Id: bindTo.BehalfUserId, Login: bindTo.BehalfLogin}
		var sch dto2.ChatDtoWithTetATet = &simpleChat{
			Id:        chat.Id,
			Name:      chat.Title,
			IsTetATet: chat.TetATet,
		}
		utils.ReplaceChatNameToLoginForTetATet(
			sch,
			&meAsUser,
			bindTo.BehalfUserId,
		)

		not.NotifyAboutCallInvitation(context.Background(), bindTo.ChatId, bindTo.UserIds, sch.GetName())

		return nil
	}
}

func CreateVideoDialStatusListener(not services.Notifications, db db.DB) VideoDialStatusListener {
	return func(msg *amqp.Delivery) error {
		data := msg.Body
		s := string(data)
		Logger.Infof("Received %v", s)

		var bindTo = new(dto2.VideoIsInvitingDto)
		err := json.Unmarshal(data, &bindTo)
		if err != nil {
			Logger.Errorf("Error during deserialize VideoIsInvitingDto %v", err)
			return nil
		}

		not.NotifyAboutDialStatus(context.Background(), bindTo.ChatId, bindTo.BehalfUserId, bindTo.Status, bindTo.UserIds)

		return nil
	}
}
