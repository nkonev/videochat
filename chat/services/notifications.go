package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/centrifugal/centrifuge"
	"github.com/getlantern/deepcopy"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"nkonev.name/chat/db"
	dto2 "nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/utils"
)

type Notifications interface {
	NotifyAboutNewChat(c echo.Context, newChatDto *dto2.ChatDtoWithAdmin, userIds []int64, tx *db.Tx)
	NotifyAboutDeleteChat(c echo.Context, chatId int64, userIds []int64, tx *db.Tx)
	NotifyAboutChangeChat(c echo.Context, chatDto *dto2.ChatDtoWithAdmin, userIds []int64, changingParticipantPage int, tx *db.Tx)

	NotifyAboutNewMessage(c echo.Context, userIds []int64, chatId int64, message *dto2.DisplayMessageDto)
	NotifyAboutDeleteMessage(c echo.Context, userIds []int64, chatId int64, message *dto2.DisplayMessageDto)
	NotifyAboutEditMessage(c echo.Context, userIds []int64, chatId int64, message *dto2.DisplayMessageDto)

	ChatNotifyMessageCount(userIds []int64, c echo.Context, chatId int64, tx *db.Tx)
	ChatNotifyAllUnreadMessageCount(userIds []int64, c echo.Context, tx *db.Tx)
	NotifyAboutMessageTyping(c echo.Context, chatId int64, user *dto2.User)
	NotifyAboutVideoCallChanged(dto dto2.ChatNotifyDto, participantIds []int64)
	NotifyAboutProfileChanged(user *dto2.User)
	NotifyAboutCallInvitation(c context.Context, chatId int64, userIds []int64, chatName string)
	NotifyAboutBroadcast(c echo.Context, chatId, userId int64, login, text string)
	NotifyAboutDialStatus(c context.Context, chatId, behalfUserId int64, status bool, usersId []int64)
}

type notifictionsImpl struct {
	centrifuge      *centrifuge.Node
	rabbitPublisher *producer.RabbitFanoutNotificationsPublisher
	db              db.DB
}

func NewNotifications(node *centrifuge.Node, rabbitPublisher *producer.RabbitFanoutNotificationsPublisher, db db.DB) Notifications {
	return &notifictionsImpl{
		centrifuge:      node,
		rabbitPublisher: rabbitPublisher,
		db:              db,
	}
}

type DisplayMessageDtoNotification struct {
	dto2.DisplayMessageDto
	ChatId int64 `json:"chatId"`
}

type UserTypingNotification struct {
	Login         string `json:"login"`
	ParticipantId int64  `json:"participantId"`
}

type VideoCallInvitation struct {
	ChatId   int64  `json:"chatId"`
	ChatName string `json:"chatName"`
}

type VideoKick struct {
	ChatId int64 `json:"chatId"`
}

type ForceMute struct {
	ChatId int64 `json:"chatId"`
}

type VideoDialChanged struct {
	UserId int64 `json:"userId"`
	Status bool  `json:"status"`
}

type VideoDialChanges struct {
	ChatId int64               `json:"chatId"`
	Dials  []*VideoDialChanged `json:"dials"`
}

const NoPagePlaceholder = -1

func (not *notifictionsImpl) NotifyAboutNewChat(c echo.Context, newChatDto *dto2.ChatDtoWithAdmin, userIds []int64, tx *db.Tx) {
	chatNotifyCommon(userIds, not, c, newChatDto, "chat_created", NoPagePlaceholder, tx)
}

func (not *notifictionsImpl) NotifyAboutChangeChat(c echo.Context, chatDto *dto2.ChatDtoWithAdmin, userIds []int64, changingParticipantPage int, tx *db.Tx) {
	chatNotifyCommon(userIds, not, c, chatDto, "chat_edited", changingParticipantPage, tx)
}

func (not *notifictionsImpl) NotifyAboutDeleteChat(c echo.Context, chatId int64, userIds []int64, tx *db.Tx) {
	chatDto := dto2.ChatDtoWithAdmin{
		BaseChatDto: dto2.BaseChatDto{
			Id: chatId,
		},
	}
	chatNotifyCommon(userIds, not, c, &chatDto, "chat_deleted", NoPagePlaceholder, tx)
}

func chatNotifyCommon(userIds []int64, not *notifictionsImpl, c echo.Context, newChatDto *dto2.ChatDtoWithAdmin, eventType string, changingParticipantPage int, tx *db.Tx) {
	for _, participantId := range userIds {
		participantChannel := utils.PersonalChannelPrefix + utils.Int64ToString(participantId)
		GetLogEntry(c.Request().Context()).Infof("Sending notification about %v the chat to participantChannel: %v", eventType, participantChannel)

		var copied *dto2.ChatDtoWithAdmin = &dto2.ChatDtoWithAdmin{}
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

		// TODO move to better place
		// see also handlers/chat.go:199 convertToDto()
		copied.CanEdit = null.BoolFrom(admin && !copied.IsTetATet)
		copied.CanDelete = null.BoolFrom(admin)
		copied.CanLeave = null.BoolFrom(!admin && !copied.IsTetATet)
		copied.UnreadMessages = unreadMessages
		copied.CanVideoKick = admin
		copied.CanAudioMute = admin
		copied.CanChangeChatAdmins = admin && !copied.IsTetATet
		copied.ParticipantsCount = newChatDto.ParticipantsCount
		copied.ChangingParticipantsPage = changingParticipantPage
		//copied.CanBroadcast = admin
		for _, participant := range copied.Participants {
			utils.ReplaceChatNameToLoginForTetATet(copied, &participant.User, participantId)
		}

		notification := dto2.CentrifugeNotification{
			Payload:   copied,
			EventType: eventType,
		}
		if marshalledBytes, err := json.Marshal(notification); err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during marshalling chat created notification: %s", err)
		} else {
			_, err := not.centrifuge.Publish(participantChannel, marshalledBytes)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error publishing to personal channel: %s", err)
			}
		}
	}
}

type ChatUnreadMessageChanged struct {
	ChatId         int64 `json:"chatId"`
	UnreadMessages int64 `json:"unreadMessages"`
}

func (not *notifictionsImpl) ChatNotifyMessageCount(userIds []int64, c echo.Context, chatId int64, tx *db.Tx) {
	for _, participantId := range userIds {
		participantChannel := utils.PersonalChannelPrefix + utils.Int64ToString(participantId)
		GetLogEntry(c.Request().Context()).Infof("Sending notification about unread messages to participantChannel: %v", participantChannel)

		unreadMessages, err := tx.GetUnreadMessagesCount(chatId, participantId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during get unread messages for userId=%v: %s", participantId, err)
			continue
		}

		payload := &ChatUnreadMessageChanged{
			ChatId:         chatId,
			UnreadMessages: unreadMessages,
		}

		notification := dto2.CentrifugeNotification{
			Payload:   payload,
			EventType: "unread_messages_changed",
		}
		if marshalledBytes, err := json.Marshal(notification); err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during marshalling chat created notification: %s", err)
		} else {
			_, err := not.centrifuge.Publish(participantChannel, marshalledBytes)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error publishing to personal channel: %s", err)
			}
		}
	}
}

func (not *notifictionsImpl) ChatNotifyAllUnreadMessageCount(userIds []int64, c echo.Context, tx *db.Tx) {
	for _, participantId := range userIds {
		participantChannel := utils.PersonalChannelPrefix + utils.Int64ToString(participantId)
		GetLogEntry(c.Request().Context()).Infof("Sending notification about all unread messages to participantChannel: %v", participantChannel)

		unreadMessages, err := tx.GetAllUnreadMessagesCount(participantId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during get all unread messages for userId=%v: %s", participantId, err)
			continue
		}

		payload := &dto2.AllUnreadMessages{
			MessagesCount: unreadMessages,
		}

		notification := dto2.CentrifugeNotification{
			Payload:   payload,
			EventType: "all_unread_messages_changed",
		}
		if marshalledBytes, err := json.Marshal(notification); err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during marshalling chat created notification: %s", err)
		} else {
			_, err := not.centrifuge.Publish(participantChannel, marshalledBytes)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error publishing to personal channel: %s", err)
			}
		}
	}
}

func messageNotifyCommon(c echo.Context, userIds []int64, chatId int64, message *dto2.DisplayMessageDto, not *notifictionsImpl, eventType string) {
	err := not.rabbitPublisher.Publish(dto2.MessageNotify{
		EventType:           eventType,
		MessageNotification: message,
	})
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
	}

	// TODO remove rest part
	// we send a notification only to those people who are currently reading the chat
	// if this is not done - when the user has many chats, he will receive many notifications and filter them on js
	activeChatUsers := []int64{}
	chatChannel := fmt.Sprintf("%v%v", utils.CHANNEL_PREFIX_CHAT_MESSAGES, chatId)
	presence, err := not.centrifuge.Presence(chatChannel)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("error during get chat presence for participantId : %s", err)
		return
	}
	for _, ci := range presence.Presence {
		if parseInt64, err := utils.ParseInt64(ci.UserID); err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during parse participantId : %s", err)
		} else {
			activeChatUsers = append(activeChatUsers, parseInt64)
		}
	}

	for _, participantId := range userIds {
		if utils.Contains(activeChatUsers, participantId) {

			participantChannel := utils.PersonalChannelPrefix + utils.Int64ToString(participantId)
			GetLogEntry(c.Request().Context()).Infof("Sending notification about create the chat to participantChannel: %v", participantChannel)

			var copied *dto2.DisplayMessageDto = &dto2.DisplayMessageDto{}
			if err := deepcopy.Copy(copied, message); err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error during performing deep copy: %s", err)
				continue
			}

			copied.CanEdit = message.OwnerId == participantId

			dn := &DisplayMessageDtoNotification{
				*copied,
				chatId,
			}
			notification := dto2.CentrifugeNotification{
				Payload:   dn,
				EventType: eventType,
			}
			if marshalledBytes, err := json.Marshal(notification); err != nil {
				GetLogEntry(c.Request().Context()).Errorf("error during marshalling chat created notification: %s", err)
			} else {
				_, err := not.centrifuge.Publish(participantChannel, marshalledBytes)
				if err != nil {
					GetLogEntry(c.Request().Context()).Errorf("error publishing to personal channel: %s", err)
				}
			}
		} else {
			GetLogEntry(c.Request().Context()).Warnf("User %v is not present in chat %v, skipping notification", participantId, chatId)
		}
	}
}

func (not *notifictionsImpl) NotifyAboutNewMessage(c echo.Context, userIds []int64, chatId int64, message *dto2.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_created")
}

func (not *notifictionsImpl) NotifyAboutDeleteMessage(c echo.Context, userIds []int64, chatId int64, message *dto2.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_deleted")
}

func (not *notifictionsImpl) NotifyAboutEditMessage(c echo.Context, userIds []int64, chatId int64, message *dto2.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_edited")
}

func (not *notifictionsImpl) NotifyAboutMessageTyping(c echo.Context, chatId int64, user *dto2.User) {
	if user == nil {
		GetLogEntry(c.Request().Context()).Errorf("user cannot be null")
		return
	}

	channelName := fmt.Sprintf("%v%v", utils.CHANNEL_PREFIX_CHAT_MESSAGES, chatId)

	ut := UserTypingNotification{
		Login:         user.Login,
		ParticipantId: user.Id,
	}

	notification := dto2.CentrifugeNotification{
		Payload:   ut,
		EventType: "user_typing",
	}

	if marshalledBytes, err := json.Marshal(notification); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("error during marshalling chat created UserTypingNotification: %s", err)
	} else {
		_, err := not.centrifuge.Publish(channelName, marshalledBytes)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error publishing to public channel: %s", err)
		}
	}
}

func (not *notifictionsImpl) NotifyAboutVideoCallChanged(cn dto2.ChatNotifyDto, participantIds []int64) {
	// TODO potential bad performance on frontend, consider batching
	for _, participantId := range participantIds {
		participantChannel := utils.PersonalChannelPrefix + utils.Int64ToString(participantId)
		Logger.Infof("Sending notification about change video chat the chat to participantChannel: %v", participantChannel)

		notification := dto2.CentrifugeNotification{
			Payload:   cn,
			EventType: "video_call_changed",
		}

		if marshalledBytes, err := json.Marshal(notification); err != nil {
			Logger.Errorf("error during marshalling chat created VideoCallChanged: %s", err)
		} else {
			_, err := not.centrifuge.Publish(participantChannel, marshalledBytes)
			if err != nil {
				Logger.Errorf("error publishing to public channel: %s", err)
			}
		}
	}
}

func (not *notifictionsImpl) NotifyAboutProfileChanged(user *dto2.User) {
	if user == nil {
		Logger.Errorf("user cannot be null")
		return
	}

	coChatters, err := not.db.GetCoChattedParticipantIdsCommon(user.Id)
	if err != nil {
		Logger.Errorf("Error during get co-chatters for %v, error: %v", user.Id, err)
	}

	for _, participantId := range coChatters {
		notification := dto2.CentrifugeNotification{
			Payload:   user,
			EventType: "user_profile_changed",
		}
		if marshalledBytes, err := json.Marshal(notification); err != nil {
			Logger.Errorf("error during marshalling user_profile_changed notification: %s", err)
		} else {
			participantChannel := utils.PersonalChannelPrefix + utils.Int64ToString(participantId)
			Logger.Infof("Sending notification about user_profile_changed to participantChannel: %v", participantChannel)
			_, err := not.centrifuge.Publish(participantChannel, marshalledBytes)
			if err != nil {
				Logger.Errorf("error publishing to personal channel: %s", err)
			}
		}
	}
}

func (not *notifictionsImpl) NotifyAboutCallInvitation(c context.Context, chatId int64, userIds []int64, chatName string) {
	notification := dto2.CentrifugeNotification{
		Payload: VideoCallInvitation{
			ChatId:   chatId,
			ChatName: chatName,
		},
		EventType: "video_call_invitation",
	}

	for _, userId := range userIds {
		participantChannel := utils.PersonalChannelPrefix + utils.Int64ToString(userId)

		if marshalledBytes, err := json.Marshal(notification); err != nil {
			GetLogEntry(c).Errorf("error during marshalling VideoCallInvitation: %s", err)
		} else {
			Logger.Infof("Sending notification about video_call_invitation to participantChannel: %v", participantChannel)
			_, err := not.centrifuge.Publish(participantChannel, marshalledBytes)
			if err != nil {
				Logger.Errorf("error publishing to personal channel: %s", err)
			}
		}
	}
}

type UserBroadcastNotification struct {
	Login  string `json:"login"`
	UserId int64  `json:"userId"`
	Text   string `json:"text"`
}

func (not *notifictionsImpl) NotifyAboutBroadcast(c echo.Context, chatId, userId int64, login, text string) {

	channelName := fmt.Sprintf("%v%v", utils.CHANNEL_PREFIX_CHAT_MESSAGES, chatId)

	ut := UserBroadcastNotification{
		Login:  login,
		UserId: userId,
		Text:   text,
	}

	notification := dto2.CentrifugeNotification{
		Payload:   ut,
		EventType: "user_broadcast",
	}

	if marshalledBytes, err := json.Marshal(notification); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("error during marshalling chat created UserBroadcastNotification: %s", err)
	} else {
		_, err := not.centrifuge.Publish(channelName, marshalledBytes)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error publishing to public channel: %s", err)
		}
	}

}

func (not *notifictionsImpl) NotifyAboutDialStatus(c context.Context, chatId, behalfUserId int64, status bool, usersIds []int64) {
	participantChannel := utils.PersonalChannelPrefix + utils.Int64ToString(behalfUserId)

	var dials = []*VideoDialChanged{}
	for _, userId := range usersIds {
		dials = append(dials, &VideoDialChanged{
			UserId: userId,
			Status: status,
		})
	}

	notification := dto2.CentrifugeNotification{
		Payload: &VideoDialChanges{
			ChatId: chatId,
			Dials:  dials,
		},
		EventType: "video_dial_status_changed",
	}

	if marshalledBytes, err := json.Marshal(notification); err != nil {
		Logger.Errorf("error during marshalling chat created VideoDialStatusChanged: %s", err)
	} else {
		_, err := not.centrifuge.Publish(participantChannel, marshalledBytes)
		if err != nil {
			Logger.Errorf("error publishing to public channel: %s", err)
		}
	}
}
