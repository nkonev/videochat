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

type Events struct {
	rabbitEventPublisher        *producer.RabbitEventsPublisher
	rabbitNotificationPublisher *producer.RabbitNotificationsPublisher
	db                          *db.DB
}

func NewEvents(rabbitEventPublisher *producer.RabbitEventsPublisher, rabbitNotificationPublisher *producer.RabbitNotificationsPublisher, db *db.DB) *Events {
	return &Events{
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

func (not *Events) NotifyAboutNewChat(c echo.Context, newChatDto *dto.ChatDtoWithAdmin, userIds []int64, tx *db.Tx) {
	chatNotifyCommon(userIds, not, c, newChatDto, "chat_created", tx)
}

func (not *Events) NotifyAboutChangeChat(c echo.Context, chatDto *dto.ChatDtoWithAdmin, userIds []int64, tx *db.Tx) {
	chatNotifyCommon(userIds, not, c, chatDto, "chat_edited", tx)
}

func (not *Events) NotifyAboutRedrawLeftChat(c echo.Context, chatDto *dto.ChatDtoWithAdmin, userId int64, tx *db.Tx) {
	chatNotifyCommon([]int64{userId}, not, c, chatDto, "chat_redraw", tx)
}

func (not *Events) NotifyAboutDeleteChat(c echo.Context, chatId int64, userIds []int64, tx *db.Tx) {
	chatDto := dto.ChatDtoWithAdmin{
		BaseChatDto: dto.BaseChatDto{
			Id: chatId,
		},
	}
	chatNotifyCommon(userIds, not, c, &chatDto, "chat_deleted", tx)
}

func chatNotifyCommon(userIds []int64, not *Events, c echo.Context, newChatDto *dto.ChatDtoWithAdmin, eventType string, tx *db.Tx) {
	GetLogEntry(c.Request().Context()).Debugf("Sending notification about %v the chat to participants: %v", eventType, userIds)

	for _, participantId := range userIds {
		if eventType == "chat_deleted" {
			err := not.rabbitEventPublisher.Publish(dto.GlobalUserEvent{
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

			isParticipant, err := tx.IsParticipant(participantId, newChatDto.Id)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error during checking is participant for userId=%v: %s", participantId, err)
				continue
			}

			unreadMessages, err := tx.GetUnreadMessagesCount(newChatDto.Id, participantId)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error during get unread messages for userId=%v: %s", participantId, err)
				continue
			}

			// see also handlers/chat.go:199 convertToDto()
			copied.SetPersonalizedFields(admin, unreadMessages, isParticipant)

			pinned, err := tx.IsChatPinned(newChatDto.Id, participantId)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error during get pinned for userId=%v: %s", participantId, err)
				continue
			}

			copied.Pinned = pinned

			for _, participant := range copied.Participants {
				utils.ReplaceChatNameToLoginForTetATet(copied, &participant.User, participantId)
			}

			err = not.rabbitEventPublisher.Publish(dto.GlobalUserEvent{
				UserId:           participantId,
				EventType:        eventType,
				ChatNotification: copied,
			})
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
			}
		}
	}
}

func (not *Events) ChatNotifyMessageCount(userIds []int64, c echo.Context, chatId int64, tx *db.Tx) {
	lastUpdated, err := tx.GetChatLastDatetimeChat(chatId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("error during get ChatLastDatetime for chat=%v: %s", chatId, err)
		return
	}

	for _, participantId := range userIds {
		GetLogEntry(c.Request().Context()).Debugf("Sending notification about unread messages to participantChannel: %v", participantId)

		unreadMessages, err := tx.GetUnreadMessagesCount(chatId, participantId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during get unread messages for userId=%v: %s", participantId, err)
			continue
		}

		payload := &dto.ChatUnreadMessageChanged{
			ChatId:             chatId,
			UnreadMessages:     unreadMessages,
			LastUpdateDateTime: lastUpdated,
		}

		err = not.rabbitEventPublisher.Publish(dto.GlobalUserEvent{
			UserId:                     participantId,
			EventType:                  "chat_unread_messages_changed",
			UnreadMessagesNotification: payload,
		})
	}
}

func messageNotifyCommon(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto, not *Events, eventType string) {

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

func (not *Events) NotifyAboutNewMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_created")
}

func (not *Events) NotifyAboutDeleteMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_deleted")
}

func (not *Events) NotifyAboutEditMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_edited")
}

func (not *Events) NotifyAboutMessageTyping(c echo.Context, chatId int64, user *dto.User) {
	if user == nil {
		GetLogEntry(c.Request().Context()).Errorf("user cannot be null")
		return
	}

	ut := dto.UserTypingNotification{
		Login:         user.Login,
		ParticipantId: user.Id,
	}

	err := not.db.IterateOverAllParticipantIds(chatId, func(participantIds []int64) error {
		for _, participantId := range participantIds {
			if participantId == user.Id {
				continue
			}

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
		return nil
	})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants")
		return
	}
}

func (not *Events) NotifyAboutProfileChanged(user *dto.User) {
	if user == nil {
		Logger.Errorf("user cannot be null")
		return
	}

	coChatters, err := not.db.GetCoChattedParticipantIdsCommon(user.Id)
	if err != nil {
		Logger.Errorf("Error during get co-chatters for %v, error: %v", user.Id, err)
	}

	for _, participantId := range coChatters {
		err = not.rabbitEventPublisher.Publish(dto.GlobalUserEvent{
			UserId:                  participantId,
			EventType:               "participant_changed",
			UserProfileNotification: user,
		})
		if err != nil {
			Logger.Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) NotifyAboutMessageBroadcast(c echo.Context, chatId, userId int64, login, text string) {
	ut := dto.MessageBroadcastNotification{
		Login:  login,
		UserId: userId,
		Text:   text,
	}

	err := not.db.IterateOverAllParticipantIds(chatId, func(participantIds []int64) error {
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
		return nil
	})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants")
		return
	}


}

func (not *Events) NotifyAddMention(c echo.Context, userIds []int64, chatId, messageId int64, message string, behalfUserId int64, behalfLogin string, chatTitle string) {
	for _, participantId := range userIds {
		err := not.rabbitNotificationPublisher.Publish(dto.NotificationEvent{
			EventType: "mention_added",
			UserId:    participantId,
			ChatId:    chatId,
			MentionNotification: &dto.MentionNotification{
				Id:   messageId,
				Text: message,
			},
			ByUserId:  behalfUserId,
			ByLogin:   behalfLogin,
			ChatTitle: chatTitle,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}

}

func (not *Events) NotifyRemoveMention(c echo.Context, userIds []int64, chatId int64, messageId int64) {
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

func (not *Events) NotifyAddReply(c echo.Context, reply *dto.ReplyDto, userId *int64, behalfUserId int64, behalfLogin string, chatTitle string) {
	if userId != nil && *userId != behalfUserId {
		err := not.rabbitNotificationPublisher.Publish(dto.NotificationEvent{
			EventType:         "reply_added",
			UserId:            *userId,
			ChatId:            reply.ChatId,
			ReplyNotification: reply,
			ByUserId:          behalfUserId,
			ByLogin:           behalfLogin,
			ChatTitle:         chatTitle,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) NotifyRemoveReply(c echo.Context, reply *dto.ReplyDto, userId *int64) {
	if userId != nil {
		err := not.rabbitNotificationPublisher.Publish(dto.NotificationEvent{
			EventType:         "reply_deleted",
			UserId:            *userId,
			ChatId:            reply.ChatId,
			ReplyNotification: reply,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) NotifyAboutNewParticipants(c echo.Context, userIds []int64, chatId int64, users []*dto.UserWithAdmin) {
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

func (not *Events) NotifyAboutDeleteParticipants(c echo.Context, userIds []int64, chatId int64, participantIdsToRemove []int64) {
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

func (not *Events) NotifyAboutChangeParticipants(c echo.Context, userIds []int64, chatId int64, participantIdsToChange []*dto.UserWithAdmin) {
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

func (not *Events) NotifyAboutPromotePinnedMessage(c echo.Context, chatId int64, msg *dto.PinnedMessageEvent, promote bool, participantIds []int64) {

	var eventType = ""
	if promote {
		eventType = "pinned_message_promote"
	} else {
		eventType = "pinned_message_unpromote"
	}

	for _, participantId := range participantIds {
		var copiedMsg = &dto.DisplayMessageDto{}
		err := deepcopy.Copy(copiedMsg, msg.Message)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during performing deep copy message: %s", err)
			return
		}

		copiedMsg.SetPersonalizedFields(participantId)

		var copiedPinnedEvent = &dto.PinnedMessageEvent{}
		err = deepcopy.Copy(copiedPinnedEvent, msg)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during performing deep copy pinned event: %s", err)
			return
		}

		copiedPinnedEvent.Message = *copiedMsg

		err = not.rabbitEventPublisher.Publish(dto.ChatEvent{
			EventType:                  eventType,
			PromoteMessageNotification: copiedPinnedEvent,
			UserId:                     participantId,
			ChatId:                     chatId,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}


func (not *Events) SendReactionEvent(c echo.Context, wasChanged bool, chatId, messageId int64, reaction string, reactionUsers []*dto.User, count int) {
	var eventType string
	if wasChanged {
		eventType = "reaction_changed"
	} else {
		eventType = "reaction_removed"
	}

	aReaction := dto.Reaction{
		Count:    int64(count),
		Reaction: reaction,
		Users: reactionUsers,
	}

	reactionChangedEvent := dto.ReactionChangedEvent{
		MessageId: messageId,
		Reaction:  aReaction,
	}

	err := not.db.IterateOverAllParticipantIds(chatId, func(participantIds []int64) error {
		for _, participantId := range participantIds {
			err := not.rabbitEventPublisher.Publish(dto.ChatEvent{
				EventType:                  eventType,
				ReactionChangedEvent: 		&reactionChangedEvent,
				UserId:                     participantId,
				ChatId:                     chatId,
			})
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
			}
		}
		return nil
	})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants")
		return
	}
}

func (not *Events) SendReactionOnYourMessage(c echo.Context, wasAdded bool, chatId, messageId int64, reaction string, behalfUserId int64, behalfLogin string, chatTitle string) {
	var eventType string
	if wasAdded {
		eventType = "reaction_notification_added"
	} else {
		eventType = "reaction_notification_removed"
	}

	_, messageOwnerId, err := not.db.GetMessageBasic(chatId, messageId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants")
		return
	}

	event := dto.ReactionEvent{
		UserId:   behalfUserId,
		Reaction: reaction,
		MessageId: messageId,
	}

	if *messageOwnerId == behalfUserId {
		return
	}
	err = not.rabbitNotificationPublisher.Publish(dto.NotificationEvent{
		EventType:                  eventType,
		ReactionEvent: 				&event,
		UserId:                     *messageOwnerId,
		ChatId:                     chatId,
		ByUserId:          behalfUserId,
		ByLogin:           behalfLogin,
		ChatTitle:         chatTitle,
	})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
	}

}
