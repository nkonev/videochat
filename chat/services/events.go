package services

import (
	"github.com/getlantern/deepcopy"
	"github.com/labstack/echo/v4"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/utils"
)

type Events interface {
	NotifyAboutNewChat(c echo.Context, newChatDto *dto.ChatDtoWithAdmin, userIds []int64, tx *db.Tx)
	NotifyAboutDeleteChat(c echo.Context, chatId int64, userIds []int64, tx *db.Tx)
	NotifyAboutChangeChat(c echo.Context, chatDto *dto.ChatDtoWithAdmin, userIds []int64, tx *db.Tx)
	NotifyAboutNewMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto)
	NotifyAboutDeleteMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto)
	NotifyAboutEditMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto)
	NotifyAboutProfileChanged(user *dto.User)
	NotifyAboutMessageTyping(c echo.Context, chatId int64, user *dto.User)
	NotifyAboutMessageBroadcast(c echo.Context, chatId, userId int64, login, text string)
	ChatNotifyMessageCount(userIds []int64, c echo.Context, chatId int64, tx *db.Tx)
	NotifyAddMention(c echo.Context, created []int64, chatId, messageId int64, message string)
	NotifyRemoveMention(c echo.Context, deleted []int64, chatId int64, messageId int64)
	NotifyAboutNewParticipants(c echo.Context, userIds []int64, chatId int64, users []*dto.UserWithAdmin)
	NotifyAboutDeleteParticipants(c echo.Context, userIds []int64, chatId int64, participantIdsToRemove []int64)
	NotifyAboutChangeParticipants(c echo.Context, userIds []int64, chatId int64, participantIdsToChange []*dto.UserWithAdmin)
}

type eventsImpl struct {
	rabbitEventPublisher        *producer.RabbitEventsPublisher
	rabbitNotificationPublisher *producer.RabbitNotificationsPublisher
	db                          *db.DB
}

func NewEvents(rabbitEventPublisher *producer.RabbitEventsPublisher, rabbitNotificationPublisher *producer.RabbitNotificationsPublisher, db *db.DB) Events {
	return &eventsImpl{
		rabbitEventPublisher:        rabbitEventPublisher,
		rabbitNotificationPublisher: rabbitNotificationPublisher,
		db:                          db,
	}
}

type DisplayMessageDtoNotification struct {
	dto.DisplayMessageDto
	ChatId int64 `json:"chatId"`
}

const NoPagePlaceholder = -1

func (not *eventsImpl) NotifyAboutNewChat(c echo.Context, newChatDto *dto.ChatDtoWithAdmin, userIds []int64, tx *db.Tx) {
	chatNotifyCommon(userIds, not, c, newChatDto, "chat_created", tx)
}

func (not *eventsImpl) NotifyAboutChangeChat(c echo.Context, chatDto *dto.ChatDtoWithAdmin, userIds []int64, tx *db.Tx) {
	chatNotifyCommon(userIds, not, c, chatDto, "chat_edited", tx)
}

func (not *eventsImpl) NotifyAboutDeleteChat(c echo.Context, chatId int64, userIds []int64, tx *db.Tx) {
	chatDto := dto.ChatDtoWithAdmin{
		BaseChatDto: dto.BaseChatDto{
			Id: chatId,
		},
	}
	chatNotifyCommon(userIds, not, c, &chatDto, "chat_deleted", tx)
}

func chatNotifyCommon(userIds []int64, not *eventsImpl, c echo.Context, newChatDto *dto.ChatDtoWithAdmin, eventType string, tx *db.Tx) {
	GetLogEntry(c.Request().Context()).Debugf("Sending notification about %v the chat to participants: %v", eventType, userIds)

	for _, participantId := range userIds {
		if eventType == "chat_deleted" {
			err := not.rabbitEventPublisher.Publish(dto.GlobalEvent{
				UserId:         participantId,
				EventType:      eventType,
				ChatDeletedDto: &dto.ChatDeletedDto{Id: newChatDto.Id},
			})
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
			}

		} else {
			var copied *dto.ChatDtoWithAdmin = &dto.ChatDtoWithAdmin{}
			if err := deepcopy.Copy(copied, newChatDto); err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error during performing deep copy: %s", err)
				continue
			}

			admin, err := tx.IsAdmin(participantId, newChatDto.Id)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error during checking is admin for userId=%v: %s", participantId, err)
				continue
			}

			unreadMessages, err := tx.GetUnreadMessagesCount(newChatDto.Id, participantId)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error during get unread messages for userId=%v: %s", participantId, err)
				continue
			}

			// see also handlers/chat.go:199 convertToDto()
			copied.SetPersonalizedFields(admin, unreadMessages)

			for _, participant := range copied.Participants {
				utils.ReplaceChatNameToLoginForTetATet(copied, &participant.User, participantId)
			}

			err = not.rabbitEventPublisher.Publish(dto.GlobalEvent{
				UserId:           participantId,
				EventType:        eventType,
				ChatNotification: newChatDto,
			})
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
			}
		}
	}
}

func (not *eventsImpl) ChatNotifyMessageCount(userIds []int64, c echo.Context, chatId int64, tx *db.Tx) {
	for _, participantId := range userIds {
		GetLogEntry(c.Request().Context()).Debugf("Sending notification about unread messages to participantChannel: %v", participantId)

		unreadMessages, err := tx.GetUnreadMessagesCount(chatId, participantId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during get unread messages for userId=%v: %s", participantId, err)
			continue
		}

		payload := &dto.ChatUnreadMessageChanged{
			ChatId:         chatId,
			UnreadMessages: unreadMessages,
		}

		err = not.rabbitEventPublisher.Publish(dto.GlobalEvent{
			UserId:                     participantId,
			EventType:                  "chat_unread_messages_changed",
			UnreadMessagesNotification: payload,
		})
	}
}

func messageNotifyCommon(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto, not *eventsImpl, eventType string) {

	for _, participantId := range userIds {
		if eventType == "message_deleted" {
			err := not.rabbitEventPublisher.Publish(dto.ChatEvent{
				EventType: eventType,
				MessageDeletedNotification: &dto.MessageDeletedDto{
					Id:     message.Id,
					ChatId: message.ChatId,
				},
				UserId: participantId,
				ChatId: chatId,
			})
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
			}
		} else {
			var copied *dto.DisplayMessageDto = &dto.DisplayMessageDto{}
			if err := deepcopy.Copy(copied, message); err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error during performing deep copy: %s", err)
				continue
			}

			copied.SetPersonalizedFields(participantId)

			err := not.rabbitEventPublisher.Publish(dto.ChatEvent{
				EventType:           eventType,
				MessageNotification: copied,
				UserId:              participantId,
				ChatId:              chatId,
			})
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
			}
		}
	}
}

func (not *eventsImpl) NotifyAboutNewMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_created")
}

func (not *eventsImpl) NotifyAboutDeleteMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_deleted")
}

func (not *eventsImpl) NotifyAboutEditMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_edited")
}

func (not *eventsImpl) NotifyAboutMessageTyping(c echo.Context, chatId int64, user *dto.User) {
	if user == nil {
		GetLogEntry(c.Request().Context()).Errorf("user cannot be null")
		return
	}

	participantIds, err := not.db.GetAllParticipantIds(chatId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants")
		return
	}

	ut := dto.UserTypingNotification{
		Login:         user.Login,
		ParticipantId: user.Id,
	}

	for _, participantId := range participantIds {
		err := not.rabbitEventPublisher.Publish(dto.ChatEvent{
			EventType:              "user_typing",
			UserTypingNotification: &ut,
			UserId:                 participantId,
			ChatId:                 chatId,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *eventsImpl) NotifyAboutProfileChanged(user *dto.User) {
	if user == nil {
		Logger.Errorf("user cannot be null")
		return
	}

	coChatters, err := not.db.GetCoChattedParticipantIdsCommon(user.Id)
	if err != nil {
		Logger.Errorf("Error during get co-chatters for %v, error: %v", user.Id, err)
	}

	for _, participantId := range coChatters {
		err = not.rabbitEventPublisher.Publish(dto.GlobalEvent{
			UserId:                  participantId,
			EventType:               "user_profile_changed",
			UserProfileNotification: user,
		})
		if err != nil {
			Logger.Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *eventsImpl) NotifyAboutMessageBroadcast(c echo.Context, chatId, userId int64, login, text string) {
	ut := dto.MessageBroadcastNotification{
		Login:  login,
		UserId: userId,
		Text:   text,
	}

	participantIds, err := not.db.GetAllParticipantIds(chatId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants")
		return
	}

	for _, participantId := range participantIds {
		err := not.rabbitEventPublisher.Publish(dto.ChatEvent{
			EventType:                    "user_broadcast",
			MessageBroadcastNotification: &ut,
			UserId:                       participantId,
			ChatId:                       chatId,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}

}

func (not *eventsImpl) NotifyAddMention(c echo.Context, userIds []int64, chatId, messageId int64, message string) {
	for _, participantId := range userIds {
		err := not.rabbitNotificationPublisher.Publish(dto.NotificationEvent{
			EventType: "mention_added",
			UserId:    participantId,
			ChatId:    chatId,
			MentionNotification: &dto.MentionNotification{
				Id:   messageId,
				Text: message,
			},
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}

}

func (not *eventsImpl) NotifyRemoveMention(c echo.Context, userIds []int64, chatId int64, messageId int64) {
	for _, participantId := range userIds {
		err := not.rabbitNotificationPublisher.Publish(dto.NotificationEvent{
			EventType: "mention_deleted",
			UserId:    participantId,
			ChatId:    chatId,
			MentionNotification: &dto.MentionNotification{
				Id: messageId,
			},
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *eventsImpl) NotifyAboutNewParticipants(c echo.Context, userIds []int64, chatId int64, users []*dto.UserWithAdmin) {
	for _, participantId := range userIds {
		err := not.rabbitEventPublisher.Publish(dto.ChatEvent{
			EventType:    "participant_added",
			UserId:       participantId,
			ChatId:       chatId,
			Participants: &users,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *eventsImpl) NotifyAboutDeleteParticipants(c echo.Context, userIds []int64, chatId int64, participantIdsToRemove []int64) {
	for _, participantId := range userIds {

		var pseudoUsers = []*dto.UserWithAdmin{}
		for _, participantIdToRemove := range participantIdsToRemove {
			pseudoUsers = append(pseudoUsers, &dto.UserWithAdmin{
				User: dto.User{Id: participantIdToRemove},
			})
		}
		err := not.rabbitEventPublisher.Publish(dto.ChatEvent{
			EventType:    "participant_deleted",
			UserId:       participantId,
			ChatId:       chatId,
			Participants: &pseudoUsers,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *eventsImpl) NotifyAboutChangeParticipants(c echo.Context, userIds []int64, chatId int64, participantIdsToChange []*dto.UserWithAdmin) {
	for _, participantId := range userIds {
		err := not.rabbitEventPublisher.Publish(dto.ChatEvent{
			EventType:    "participant_edited",
			UserId:       participantId,
			ChatId:       chatId,
			Participants: &participantIdsToChange,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}
