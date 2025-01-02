package services

import (
	"context"
	"fmt"
	"github.com/getlantern/deepcopy"
	"github.com/guregu/null"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/utils"
	"time"
)

type Events struct {
	rabbitEventPublisher        *producer.RabbitEventsPublisher
	rabbitNotificationPublisher *producer.RabbitNotificationsPublisher
	tr                          trace.Tracer
	lgr                         *logger.Logger
}

func NewEvents(rabbitEventPublisher *producer.RabbitEventsPublisher, rabbitNotificationPublisher *producer.RabbitNotificationsPublisher, lgr *logger.Logger) *Events {
	tr := otel.Tracer("event")

	return &Events{
		rabbitEventPublisher:        rabbitEventPublisher,
		rabbitNotificationPublisher: rabbitNotificationPublisher,
		tr:                          tr,
		lgr:                         lgr,
	}
}

type DisplayMessageDtoNotification struct {
	dto.DisplayMessageDto
	ChatId int64 `json:"chatId"`
}

const NoPagePlaceholder = -1

func (not *Events) NotifyAboutNewChat(ctx context.Context, newChatDto *dto.ChatDto, userIds []int64, isSingleParticipant bool, overrideIsParticipant bool, tx *db.Tx, areAdminsMap map[int64]bool) {
	not.chatNotifyCommon(ctx, userIds, newChatDto, "chat_created", isSingleParticipant, overrideIsParticipant, tx, areAdminsMap)
}

func (not *Events) NotifyAboutChangeChat(ctx context.Context, chatDto *dto.ChatDto, userIds []int64, isSingleParticipant bool, overrideIsParticipant bool, tx *db.Tx, areAdminsMap map[int64]bool) {
	not.chatNotifyCommon(ctx, userIds, chatDto, "chat_edited", isSingleParticipant, overrideIsParticipant, tx, areAdminsMap)
}

func (not *Events) NotifyAboutRedrawLeftChat(ctx context.Context, chatDto *dto.ChatDto, userId int64, isSingleParticipant bool, overrideIsParticipant bool, tx *db.Tx, areAdminsMap map[int64]bool) {
	not.chatNotifyCommon(ctx, []int64{userId}, chatDto, "chat_redraw", isSingleParticipant, overrideIsParticipant, tx, areAdminsMap)
}

func (not *Events) NotifyAboutDeleteChat(ctx context.Context, chatId int64, userIds []int64, tx *db.Tx) {
	chatDto := dto.ChatDto{
		BaseChatDto: dto.BaseChatDto{
			Id: chatId,
		},
	}
	not.chatNotifyCommon(ctx, userIds, &chatDto, "chat_deleted", false, false, tx, nil)
}

/**
 * isSingleParticipant should be taken from responseDto or count. using len(participants) where participants are a portion from Iterate...() is incorrect because we can get only one user in the last iteration
 */
func (not *Events) chatNotifyCommon(ctx context.Context, userIds []int64, newChatDto *dto.ChatDto, eventType string, isSingleParticipant bool, overrideIsParticipant bool, tx *db.Tx, areAdminsMap map[int64]bool) {
	not.lgr.WithTracing(ctx).Debugf("Sending notification about %v the chat to participants: %v", eventType, userIds)

	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	if eventType == "chat_deleted" {
		for _, participantId := range userIds {
			err := not.rabbitEventPublisher.Publish(ctx, dto.GlobalUserEvent{
				UserId:         participantId,
				EventType:      eventType,
				ChatDeletedDto: &dto.ChatDeletedDto{Id: newChatDto.Id},
			})
			if err != nil {
				not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
			}
		}
	} else {

		unreadMessages, err := tx.GetUnreadMessagesCountBatchByParticipants(ctx, userIds, newChatDto.Id)
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("error during get unread messages: %v", err)
			return
		}

		isChatPinnedMap, err := tx.IsChatPinnedBatch(ctx, userIds, newChatDto.Id)
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("error during get pinned: %v", err)
			return
		}

		for _, participantId := range userIds {
			var copied *dto.ChatDto = &dto.ChatDto{}
			if err := deepcopy.Copy(copied, newChatDto); err != nil {
				not.lgr.WithTracing(ctx).Errorf("error during performing deep copy: %s", err)
				continue
			}

			// see also handlers/chat.go:199 convertToDto()
			copied.SetPersonalizedFields(areAdminsMap[participantId], unreadMessages[participantId], overrideIsParticipant)

			copied.Pinned = isChatPinnedMap[participantId]

			for _, participant := range copied.Participants {
				utils.ReplaceChatNameToLoginForTetATet(copied, participant, participantId, isSingleParticipant)
			}

			err = not.rabbitEventPublisher.Publish(ctx, dto.GlobalUserEvent{
				UserId:           participantId,
				EventType:        eventType,
				ChatNotification: copied,
			})
			if err != nil {
				not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
			}
		}
	}
}

func (not *Events) ChatNotifyMessageCount(ctx context.Context, userIds []int64, chatId int64, tx *db.Tx) {

	lastUpdated, err := tx.GetChatLastDatetimeChat(ctx, chatId)
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("error during get ChatLastDatetime for chat=%v: %s", chatId, err)
		return
	}

	unreadMessagesByUserId, err := tx.GetUnreadMessagesCountBatchByParticipants(ctx, userIds, chatId)
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("error during get GetUnreadMessagesCountBatchByParticipants for chat=%v: %v", chatId, err)
		return
	}

	for _, participantId := range userIds {
		not.lgr.WithTracing(ctx).Debugf("Sending notification about unread messages to participantChannel: %v", participantId)

		not.NotifyAboutUnreadMessage(ctx, chatId, participantId, unreadMessagesByUserId[participantId], lastUpdated)
	}
}

func (not *Events) NotifyAboutUnreadMessage(ctx context.Context, chatId int64, participantId int64, unreadMessages int64, lastUpdateDateTime time.Time) {
	eventType := "chat_unread_messages_changed"
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("global.user.%s", eventType))
	defer messageSpan.End()

	payload := &dto.ChatUnreadMessageChanged{
		ChatId:             chatId,
		UnreadMessages:     unreadMessages,
		LastUpdateDateTime: lastUpdateDateTime,
	}

	err := not.rabbitEventPublisher.Publish(ctx, dto.GlobalUserEvent{
		UserId:                     participantId,
		EventType:                  eventType,
		UnreadMessagesNotification: payload,
	})
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("Error during sending: %v", err)
	}
}

func (not *Events) NotifyAboutHasNewMessagesChanged(ctx context.Context, participantId int64, hasNewMessages bool) {
	eventType := "has_unread_messages_changed"
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("global.user.%s", eventType))
	defer messageSpan.End()

	err := not.rabbitEventPublisher.Publish(ctx, dto.GlobalUserEvent{
		UserId:    participantId,
		EventType: eventType,
		HasUnreadMessagesChanged: &dto.HasUnreadMessagesChanged{
			HasUnreadMessages: hasNewMessages,
		},
	})
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
	}
}

func (not *Events) messageNotifyCommon(ctx context.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto, eventType string, chatBasic *db.BasicChatDto, chatAdmins map[int64]bool) {
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("message.%s", eventType))
	defer messageSpan.End()

	for _, participantId := range userIds {
		if eventType == "message_deleted" {
			err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
				EventType: eventType,
				MessageDeletedNotification: &dto.MessageDeletedDto{
					Id:     message.Id,
					ChatId: message.ChatId,
				},
				UserId: participantId,
				ChatId: chatId,
			})
			if err != nil {
				not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
			}
		} else {
			var copied *dto.DisplayMessageDto = &dto.DisplayMessageDto{}
			if err := deepcopy.Copy(copied, message); err != nil {
				not.lgr.WithTracing(ctx).Errorf("error during performing deep copy: %s", err)
				continue
			}

			copied.SetPersonalizedFields(chatBasic.RegularParticipantCanPublishMessage, chatBasic.RegularParticipantCanPinMessage, chatAdmins[participantId], participantId)

			err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
				EventType:           eventType,
				MessageNotification: copied,
				UserId:              participantId,
				ChatId:              chatId,
			})
			if err != nil {
				not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
			}
		}
	}
}

func (not *Events) NotifyAboutNewMessage(ctx context.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto, chatBasic *db.BasicChatDto, chatAdmins map[int64]bool) {
	not.messageNotifyCommon(ctx, userIds, chatId, message, "message_created", chatBasic, chatAdmins)
}

func (not *Events) NotifyAboutDeleteMessage(ctx context.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto) {
	not.messageNotifyCommon(ctx, userIds, chatId, message, "message_deleted", nil, nil)
}

func (not *Events) NotifyAboutEditMessage(ctx context.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto, chatBasic *db.BasicChatDto, chatAdmins map[int64]bool) {
	not.messageNotifyCommon(ctx, userIds, chatId, message, "message_edited", chatBasic, chatAdmins)
}

func (not *Events) NotifyAboutMessageTyping(ctx context.Context, chatId int64, user *dto.User, co db.CommonOperations) {
	if user == nil {
		not.lgr.WithTracing(ctx).Errorf("user cannot be null")
		return
	}

	eventType := "user_typing"
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	ut := dto.UserTypingNotification{
		Login:         user.Login,
		ParticipantId: user.Id,
	}

	err := co.IterateOverChatParticipantIds(ctx, chatId, func(participantIds []int64) error {
		for _, participantId := range participantIds {
			if participantId == user.Id {
				continue
			}

			err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
				EventType:              eventType,
				UserTypingNotification: &ut,
				UserId:                 participantId,
				ChatId:                 chatId,
			})
			if err != nil {
				not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
			}
		}
		return nil
	})
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("Error during getting chat participants")
		return
	}
}

func (not *Events) NotifyAboutProfileChanged(ctx context.Context, user *dto.User, co db.CommonOperations) {
	if user == nil {
		not.lgr.WithTracing(ctx).Errorf("user cannot be null")
		return
	}

	eventType := "participant_changed"
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("global.user.%s", eventType))
	defer messageSpan.End()

	err := co.IterateOverCoChattedParticipantIds(ctx, user.Id, func(participantIds []int64) error {
		var internalErr error
		for _, participantId := range participantIds {
			internalErr = not.rabbitEventPublisher.Publish(ctx, dto.GlobalUserEvent{
				UserId:                           participantId,
				EventType:                        eventType,
				CoChattedParticipantNotification: user,
			})
		}
		return internalErr
	})
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("Error during get co-chatters for %v, error: %v", user.Id, err)
	}
}

func (not *Events) NotifyAboutMessageBroadcast(ctx context.Context, chatId, userId int64, login, text string, co db.CommonOperations) {
	eventType := "user_broadcast"
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	ut := dto.MessageBroadcastNotification{
		Login:  login,
		UserId: userId,
		Text:   text,
	}

	err := co.IterateOverChatParticipantIds(ctx, chatId, func(participantIds []int64) error {
		for _, participantId := range participantIds {
			err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
				EventType:                    eventType,
				MessageBroadcastNotification: &ut,
				UserId:                       participantId,
				ChatId:                       chatId,
			})
			if err != nil {
				not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
			}
		}
		return nil
	})
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("Error during getting chat participants")
		return
	}
}

func (not *Events) NotifyAddMention(ctx context.Context, userIds []int64, chatId, messageId int64, message string, behalfUserId int64, behalfLogin string, behalfAvatar *string, chatTitle string) {
	eventType := "mention_added"
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("notification.%s", eventType))
	defer messageSpan.End()

	for _, participantId := range userIds {
		err := not.rabbitNotificationPublisher.Publish(ctx, dto.NotificationEvent{
			EventType: eventType,
			UserId:    participantId,
			ChatId:    chatId,
			MentionNotification: &dto.MentionNotification{
				Id:   messageId,
				Text: message,
			},
			ByUserId:  behalfUserId,
			ByLogin:   behalfLogin,
			ByAvatar:  behalfAvatar,
			ChatTitle: chatTitle,
		})
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}

}

func (not *Events) NotifyRemoveMention(ctx context.Context, userIds []int64, chatId int64, messageId int64) {
	eventType := "mention_deleted"
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("notification.%s", eventType))
	defer messageSpan.End()

	for _, participantId := range userIds {
		err := not.rabbitNotificationPublisher.Publish(ctx, dto.NotificationEvent{
			EventType: eventType,
			UserId:    participantId,
			ChatId:    chatId,
			MentionNotification: &dto.MentionNotification{
				Id: messageId,
			},
		})
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) NotifyAddReply(ctx context.Context, reply *dto.ReplyDto, userId *int64, behalfUserId int64, behalfLogin string, behalfAvatar *string, chatTitle string) {
	if userId != nil && *userId != behalfUserId {
		eventType := "reply_added"
		ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("notification.%s", eventType))
		defer messageSpan.End()

		err := not.rabbitNotificationPublisher.Publish(ctx, dto.NotificationEvent{
			EventType:         eventType,
			UserId:            *userId,
			ChatId:            reply.ChatId,
			ReplyNotification: reply,
			ByUserId:          behalfUserId,
			ByLogin:           behalfLogin,
			ByAvatar:          behalfAvatar,
			ChatTitle:         chatTitle,
		})
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) NotifyRemoveReply(ctx context.Context, reply *dto.ReplyDto, userId *int64) {
	if userId != nil {
		eventType := "reply_deleted"
		ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("notification.%s", eventType))
		defer messageSpan.End()

		err := not.rabbitNotificationPublisher.Publish(ctx, dto.NotificationEvent{
			EventType:         eventType,
			UserId:            *userId,
			ChatId:            reply.ChatId,
			ReplyNotification: reply,
		})
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) NotifyAboutNewParticipants(ctx context.Context, userIds []int64, chatId int64, users []*dto.UserWithAdmin) {
	eventType := "participant_added"
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	for _, participantId := range userIds {
		err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
			EventType:    eventType,
			UserId:       participantId,
			ChatId:       chatId,
			Participants: &users,
		})
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) NotifyAboutDeleteParticipants(ctx context.Context, userIds []int64, chatId int64, participantIdsToRemove []int64) {
	eventType := "participant_deleted"
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	for _, participantId := range userIds {

		var pseudoUsers = []*dto.UserWithAdmin{}
		for _, participantIdToRemove := range participantIdsToRemove {
			pseudoUsers = append(pseudoUsers, &dto.UserWithAdmin{
				User: dto.User{Id: participantIdToRemove},
			})
		}
		err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
			EventType:    eventType,
			UserId:       participantId,
			ChatId:       chatId,
			Participants: &pseudoUsers,
		})
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) NotifyAboutChangeParticipants(ctx context.Context, userIds []int64, chatId int64, participantIdsToChange []*dto.UserWithAdmin) {
	eventType := "participant_edited"
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	for _, participantId := range userIds {
		err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
			EventType:    eventType,
			UserId:       participantId,
			ChatId:       chatId,
			Participants: &participantIdsToChange,
		})
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) NotifyAboutPromotePinnedMessage(ctx context.Context, chatId int64, msg *dto.PinnedMessageEvent, promote bool, participantId int64) {

	var eventType = ""
	if promote {
		eventType = "pinned_message_promote"
	} else {
		eventType = "pinned_message_unpromote"
	}

	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
		EventType:                  eventType,
		PromoteMessageNotification: msg,
		UserId:                     participantId,
		ChatId:                     chatId,
	})
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
	}
}

func (not *Events) NotifyAboutPromotePinnedMessageEdit(ctx context.Context, chatId int64, msg *dto.PinnedMessageEvent, participantId int64) {

	var eventType = "pinned_message_edit"

	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
		EventType:                  eventType,
		PromoteMessageNotification: msg,
		UserId:                     participantId,
		ChatId:                     chatId,
	})
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
	}
}

func (not *Events) NotifyAboutPublishedMessage(ctx context.Context, chatId int64, msg *dto.PublishedMessageEvent, publish bool, participantIds []int64, regularParticipantCanPublishMessage bool, areAdmins map[int64]bool) {

	var eventType = ""
	if publish {
		eventType = "published_message_add"
	} else {
		eventType = "published_message_remove"
	}

	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	for _, participantId := range participantIds {

		var copied *dto.PublishedMessageEvent = &dto.PublishedMessageEvent{}
		if err := deepcopy.Copy(copied, msg); err != nil {
			not.lgr.WithTracing(ctx).Errorf("error during performing deep copy: %s", err)
			continue
		}

		copied.Message.CanPublish = dto.CanPublishMessage(regularParticipantCanPublishMessage, areAdmins[participantId], copied.Message.OwnerId, participantId)

		err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
			EventType:                    eventType,
			PublishedMessageNotification: copied,
			UserId:                       participantId,
			ChatId:                       chatId,
		})
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) NotifyAboutPublishedMessageEdit(ctx context.Context, chatId int64, msg *dto.PublishedMessageEvent, participantIds []int64, regularParticipantCanPublishMessage bool, areAdmins map[int64]bool) {

	var eventType = "published_message_edit"

	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	for _, participantId := range participantIds {

		var copied *dto.PublishedMessageEvent = &dto.PublishedMessageEvent{}
		if err := deepcopy.Copy(copied, msg); err != nil {
			not.lgr.WithTracing(ctx).Errorf("error during performing deep copy: %s", err)
			continue
		}

		copied.Message.CanPublish = dto.CanPublishMessage(regularParticipantCanPublishMessage, areAdmins[participantId], copied.Message.OwnerId, participantId)

		err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
			EventType:                    eventType,
			PublishedMessageNotification: copied,
			UserId:                       participantId,
			ChatId:                       chatId,
		})
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *Events) SendReactionEvent(ctx context.Context, wasChanged bool, chatId, messageId int64, reaction string, reactionUsers []*dto.User, count int, tx *db.Tx) {
	var eventType string
	if wasChanged {
		eventType = "reaction_changed"
	} else {
		eventType = "reaction_removed"
	}

	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("notification.%s", eventType))
	defer messageSpan.End()

	aReaction := dto.Reaction{
		Count:    int64(count),
		Reaction: reaction,
		Users:    reactionUsers,
	}

	reactionChangedEvent := dto.ReactionChangedEvent{
		MessageId: messageId,
		Reaction:  aReaction,
	}

	err := tx.IterateOverChatParticipantIds(ctx, chatId, func(participantIds []int64) error {
		for _, participantId := range participantIds {
			err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
				EventType:            eventType,
				ReactionChangedEvent: &reactionChangedEvent,
				UserId:               participantId,
				ChatId:               chatId,
			})
			if err != nil {
				not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
			}
		}
		return nil
	})
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("Error during getting chat participants")
		return
	}
}

func (not *Events) SendReactionOnYourMessage(ctx context.Context, wasAdded bool, chatId, messageId, messageOwnerId int64, reaction string, behalfUserId int64, behalfLogin string, behalfAvatar *string, chatTitle string) {
	var eventType string
	if wasAdded {
		eventType = "reaction_notification_added"
	} else {
		eventType = "reaction_notification_removed"
	}

	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("notification.%s", eventType))
	defer messageSpan.End()

	event := dto.ReactionEvent{
		UserId:    behalfUserId,
		Reaction:  reaction,
		MessageId: messageId,
	}

	err := not.rabbitNotificationPublisher.Publish(ctx, dto.NotificationEvent{
		EventType:     eventType,
		ReactionEvent: &event,
		UserId:        messageOwnerId,
		ChatId:        chatId,
		ByUserId:      behalfUserId,
		ByLogin:       behalfLogin,
		ByAvatar:      behalfAvatar,
		ChatTitle:     chatTitle,
	})
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
	}

}

func (not *Events) NotifyMessagesReloadCommand(ctx context.Context, chatId int64, participantIds []int64) {
	eventType := "messages_reload"
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	for _, participantId := range participantIds {
		err := not.rabbitEventPublisher.Publish(ctx, dto.ChatEvent{
			EventType: eventType,
			UserId:    participantId,
			ChatId:    chatId,
		})
		if err != nil {
			not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}

}

func (not *Events) NotifyNewMessageBrowserNotification(ctx context.Context, add bool, participantId int64, chatId int64, chatName string, chatAvatar null.String, messageId int64, messageText string, ownerId int64, ownerLogin string) {
	eventType := "browser_notification_add_message"
	if !add {
		eventType = "browser_notification_remove_message"
	}

	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("notification.%s", eventType))
	defer messageSpan.End()

	err := not.rabbitEventPublisher.Publish(ctx, dto.GlobalUserEvent{
		UserId:    participantId,
		EventType: eventType,
		BrowserNotification: &dto.BrowserNotification{
			ChatId:      chatId,
			ChatName:    chatName,
			ChatAvatar:  chatAvatar.Ptr(),
			MessageId:   messageId,
			MessageText: messageText,
			OwnerId:     ownerId,
			OwnerLogin:  ownerLogin,
		},
	})
	if err != nil {
		not.lgr.WithTracing(ctx).Errorf("Error during sending to rabbitmq : %s", err)
	}

}
