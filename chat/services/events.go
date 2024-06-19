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
}

func NewEvents(rabbitEventPublisher *producer.RabbitEventsPublisher, rabbitNotificationPublisher *producer.RabbitNotificationsPublisher) *Events {
	return &Events{
		rabbitEventPublisher:        rabbitEventPublisher,
		rabbitNotificationPublisher: rabbitNotificationPublisher,
	}
}

type DisplayMessageDtoNotification struct {
	dto.DisplayMessageDto
	ChatId int64 `json:"chatId"`
}

const NoPagePlaceholder = -1

func (not *Events) NotifyAboutNewChat(c echo.Context, newChatDto *dto.ChatDtoWithAdmin, userIds []int64, isSingleParticipant bool, overrideIsParticipant bool, tx *db.Tx, areAdminsMap map[int64]bool) {
	chatNotifyCommon(userIds, not, c, newChatDto, "chat_created", isSingleParticipant, overrideIsParticipant, tx, areAdminsMap)
}

func (not *Events) NotifyAboutChangeChat(c echo.Context, chatDto *dto.ChatDtoWithAdmin, userIds []int64,isSingleParticipant bool, overrideIsParticipant bool, tx *db.Tx, areAdminsMap map[int64]bool) {
	chatNotifyCommon(userIds, not, c, chatDto, "chat_edited", isSingleParticipant, overrideIsParticipant, tx, areAdminsMap)
}

func (not *Events) NotifyAboutRedrawLeftChat(c echo.Context, chatDto *dto.ChatDtoWithAdmin, userId int64,isSingleParticipant bool, overrideIsParticipant bool, tx *db.Tx, areAdminsMap map[int64]bool) {
	chatNotifyCommon([]int64{userId}, not, c, chatDto, "chat_redraw", isSingleParticipant, overrideIsParticipant, tx, areAdminsMap)
}

func (not *Events) NotifyAboutDeleteChat(c echo.Context, chatId int64, userIds []int64, tx *db.Tx) {
	chatDto := dto.ChatDtoWithAdmin{
		BaseChatDto: dto.BaseChatDto{
			Id: chatId,
		},
	}
	chatNotifyCommon(userIds, not, c, &chatDto, "chat_deleted", false, false, tx, nil)
}

/**
 * isSingleParticipant should be taken from responseDto or count. using len(participants) where participants are a portion from Iterate...() is incorrect because we can get only one user in the last iteration
 */
func chatNotifyCommon(userIds []int64, not *Events, c echo.Context, newChatDto *dto.ChatDtoWithAdmin, eventType string, isSingleParticipant bool, overrideIsParticipant bool, tx *db.Tx, areAdminsMap map[int64]bool) {
	GetLogEntry(c.Request().Context()).Debugf("Sending notification about %v the chat to participants: %v", eventType, userIds)

	if eventType == "chat_deleted" {
		for _, participantId := range userIds {
			err := not.rabbitEventPublisher.Publish(dto.GlobalUserEvent{
				UserId:         participantId,
				EventType:      eventType,
				ChatDeletedDto: &dto.ChatDeletedDto{Id: newChatDto.Id},
			})
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
			}
		}
	} else {

		unreadMessages, err := tx.GetUnreadMessagesCountBatchByParticipants(userIds, newChatDto.Id)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during get unread messages: %v", err)
			return
		}

		isChatPinnedMap, err := tx.IsChatPinnedBatch(userIds, newChatDto.Id)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during get pinned: %v", err)
			return
		}

		for _, participantId := range userIds {
			var copied *dto.ChatDtoWithAdmin = &dto.ChatDtoWithAdmin{}
			if err := deepcopy.Copy(copied, newChatDto); err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error during performing deep copy: %s", err)
				continue
			}

			// see also handlers/chat.go:199 convertToDto()
			copied.SetPersonalizedFields(areAdminsMap[participantId], unreadMessages[participantId], overrideIsParticipant)

			copied.Pinned = isChatPinnedMap[participantId]

			for _, participant := range copied.Participants {
				utils.ReplaceChatNameToLoginForTetATet(copied, &participant.User, participantId, isSingleParticipant)
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

	unreadMessagesByUserId, err := tx.GetUnreadMessagesCountBatchByParticipants(userIds, chatId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("error during get GetUnreadMessagesCountBatchByParticipants for chat=%v: %v", chatId, err)
		return
	}

	for _, participantId := range userIds {
		GetLogEntry(c.Request().Context()).Debugf("Sending notification about unread messages to participantChannel: %v", participantId)

		payload := &dto.ChatUnreadMessageChanged{
			ChatId:             chatId,
			UnreadMessages:     unreadMessagesByUserId[participantId],
			LastUpdateDateTime: lastUpdated,
		}

		err = not.rabbitEventPublisher.Publish(dto.GlobalUserEvent{
			UserId:                     participantId,
			EventType:                  "chat_unread_messages_changed",
			UnreadMessagesNotification: payload,
		})
	}
}

func (not *Events) NotifyAboutHasNewMessagesChanged(c echo.Context, participantId int64, hasNewMessages bool) {
	err := not.rabbitEventPublisher.Publish(dto.GlobalUserEvent{
		UserId:                     participantId,
		EventType:                  "has_unread_messages_changed",
		HasUnreadMessagesChanged:   &dto.HasUnreadMessagesChanged{
			HasUnreadMessages:      hasNewMessages,
		},
	})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
	}
}

func messageNotifyCommon(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto, not *Events, eventType string, chatRegularParticipantCanPublishMessage bool, chatAdmins map[int64]bool) {

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

			copied.SetPersonalizedFields(chatRegularParticipantCanPublishMessage, chatAdmins[participantId], participantId)

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

func (not *Events) NotifyAboutNewMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto, chatRegularParticipantCanPublishMessage bool, chatAdmins map[int64]bool) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_created", chatRegularParticipantCanPublishMessage, chatAdmins)
}

func (not *Events) NotifyAboutDeleteMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_deleted", false, nil)
}

func (not *Events) NotifyAboutEditMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto, chatRegularParticipantCanPublishMessage bool, chatAdmins map[int64]bool) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_edited", chatRegularParticipantCanPublishMessage, chatAdmins)
}

func (not *Events) NotifyAboutMessageTyping(c echo.Context, chatId int64, user *dto.User, co db.CommonOperations) {
	if user == nil {
		GetLogEntry(c.Request().Context()).Errorf("user cannot be null")
		return
	}

	ut := dto.UserTypingNotification{
		Login:         user.Login,
		ParticipantId: user.Id,
	}

	err := co.IterateOverChatParticipantIds(chatId, func(participantIds []int64) error {
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

func (not *Events) NotifyAboutProfileChanged(user *dto.User, co db.CommonOperations) {
	if user == nil {
		Logger.Errorf("user cannot be null")
		return
	}

	err := co.IterateOverCoChattedParticipantIds(user.Id, func(participantIds []int64) error {
		var internalErr error
		for _, participantId := range participantIds {
			internalErr = not.rabbitEventPublisher.Publish(dto.GlobalUserEvent{
				UserId:                  participantId,
				EventType:               "participant_changed",
				CoChattedParticipantNotification: user,
			})
		}
		return internalErr
	})
	if err != nil {
		Logger.Errorf("Error during get co-chatters for %v, error: %v", user.Id, err)
	}
}

func (not *Events) NotifyAboutMessageBroadcast(c echo.Context, chatId, userId int64, login, text string, co db.CommonOperations) {
	ut := dto.MessageBroadcastNotification{
		Login:  login,
		UserId: userId,
		Text:   text,
	}

	err := co.IterateOverChatParticipantIds(chatId, func(participantIds []int64) error {
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

		err := not.rabbitEventPublisher.Publish(dto.ChatEvent{
			EventType:                  eventType,
			PromoteMessageNotification: msg,
			UserId:                     participantId,
			ChatId:                     chatId,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) NotifyAboutPublishedMessage(c echo.Context, chatId int64, msg *dto.PublishedMessageEvent, publish bool, participantIds []int64, regularParticipantCanPublishMessage bool, areAdmins map[int64]bool) {

	var eventType = ""
	if publish {
		eventType = "published_message_add"
	} else {
		eventType = "published_message_remove"
	}

	for _, participantId := range participantIds {

		var copied *dto.PublishedMessageEvent = &dto.PublishedMessageEvent{}
		if err := deepcopy.Copy(copied, msg); err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during performing deep copy: %s", err)
			continue
		}

		copied.Message.CanPublish = dto.CanPublishMessage(regularParticipantCanPublishMessage, areAdmins[participantId], copied.Message.OwnerId, participantId)

		err := not.rabbitEventPublisher.Publish(dto.ChatEvent{
			EventType:                    eventType,
			PublishedMessageNotification: copied,
			UserId:                       participantId,
			ChatId:                       chatId,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) SendReactionEvent(c echo.Context, wasChanged bool, chatId, messageId int64, reaction string, reactionUsers []*dto.User, count int, tx *db.Tx) {
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

	err := tx.IterateOverChatParticipantIds(chatId, func(participantIds []int64) error {
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

func (not *Events) SendReactionOnYourMessage(c echo.Context, wasAdded bool, chatId, messageId, messageOwnerId int64, reaction string, behalfUserId int64, behalfLogin string, chatTitle string) {
	var eventType string
	if wasAdded {
		eventType = "reaction_notification_added"
	} else {
		eventType = "reaction_notification_removed"
	}

	event := dto.ReactionEvent{
		UserId:   behalfUserId,
		Reaction: reaction,
		MessageId: messageId,
	}

	if messageOwnerId == behalfUserId {
		return
	}
	err := not.rabbitNotificationPublisher.Publish(dto.NotificationEvent{
		EventType:                  eventType,
		ReactionEvent: 				&event,
		UserId:                     messageOwnerId,
		ChatId:                     chatId,
		ByUserId:          behalfUserId,
		ByLogin:           behalfLogin,
		ChatTitle:         chatTitle,
	})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
	}

}

func (not *Events) NotifyMessagesReloadCommand(c echo.Context, chatId int64, participantIds []int64) {
	for _, participantId := range participantIds {
		err := not.rabbitEventPublisher.Publish(dto.ChatEvent{
			EventType:                  "messages_reload",
			UserId:                     participantId,
			ChatId:                     chatId,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}

}
