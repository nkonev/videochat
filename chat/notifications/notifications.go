package notifications

import (
	"encoding/json"
	"fmt"
	"github.com/centrifugal/centrifuge"
	"github.com/getlantern/deepcopy"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

type Notifications interface {
	NotifyAboutNewChat(c echo.Context, newChatDto *dto.ChatDto, userIds []int64, tx *db.Tx)
	NotifyAboutDeleteChat(c echo.Context, chatDto *dto.ChatDto, userIds []int64, tx *db.Tx)
	NotifyAboutChangeChat(c echo.Context, chatDto *dto.ChatDto, userIds []int64, tx *db.Tx)
	NotifyAboutNewMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto)
	NotifyAboutDeleteMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto)
	NotifyAboutEditMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto)
	ChatNotifyMessageCount(userIds []int64, c echo.Context, chatId int64, tx *db.Tx)
	NotifyAboutMessageTyping(c echo.Context, chatId int64, user *dto.User)
}

type notifictionsImpl struct {
	centrifuge *centrifuge.Node
}

func NewNotifications(node *centrifuge.Node) Notifications {
	return &notifictionsImpl{
		centrifuge: node,
	}
}

// created or modified
type CentrifugeNotification struct {
	Payload   interface{} `json:"payload"`
	EventType string      `json:"type"`
}

type DisplayMessageDtoNotification struct {
	dto.DisplayMessageDto
	ChatId int64 `json:"chatId"`
}

type UserTypingNotification struct {
	Login         string `json:"login"`
	ParticipantId int64  `json:"participantId"`
}

func (not *notifictionsImpl) NotifyAboutNewChat(c echo.Context, newChatDto *dto.ChatDto, userIds []int64, tx *db.Tx) {
	chatNotifyCommon(userIds, not, c, newChatDto, "chat_created", tx)
}

func (not *notifictionsImpl) NotifyAboutChangeChat(c echo.Context, chatDto *dto.ChatDto, userIds []int64, tx *db.Tx) {
	chatNotifyCommon(userIds, not, c, chatDto, "chat_edited", tx)
}

func (not *notifictionsImpl) NotifyAboutDeleteChat(c echo.Context, chatDto *dto.ChatDto, userIds []int64, tx *db.Tx) {
	chatNotifyCommon(userIds, not, c, chatDto, "chat_deleted", tx)
}

func chatNotifyCommon(userIds []int64, not *notifictionsImpl, c echo.Context, newChatDto *dto.ChatDto, eventType string, tx *db.Tx) {
	for _, participantId := range userIds {
		participantChannel := not.centrifuge.PersonalChannel(utils.Int64ToString(participantId))
		GetLogEntry(c.Request()).Infof("Sending notification about create the chat to participantChannel: %v", participantChannel)

		var copied *dto.ChatDto = &dto.ChatDto{}
		if err := deepcopy.Copy(copied, newChatDto); err != nil {
			GetLogEntry(c.Request()).Errorf("error during performing deep copy: %s", err)
			continue
		}

		admin, err := tx.IsAdmin(participantId, newChatDto.Id)
		if err != nil {
			GetLogEntry(c.Request()).Errorf("error during checking is admin for userId=%v: %s", participantId, err)
			continue
		}

		unreadMessages, err := tx.GetUnreadMessages(newChatDto.Id, participantId)
		if err != nil {
			GetLogEntry(c.Request()).Errorf("error during get unread messages for userId=%v: %s", participantId, err)
			continue
		}

		// TODO move to better place
		copied.CanEdit = null.BoolFrom(admin)
		copied.CanLeave = null.BoolFrom(!admin)
		copied.UnreadMessages = unreadMessages

		notification := CentrifugeNotification{
			Payload:   copied,
			EventType: eventType,
		}
		if marshalledBytes, err2 := json.Marshal(notification); err2 != nil {
			GetLogEntry(c.Request()).Errorf("error during marshalling chat created notification: %s", err2)
		} else {
			_, err := not.centrifuge.Publish(participantChannel, marshalledBytes)
			if err != nil {
				GetLogEntry(c.Request()).Errorf("error publishing to personal channel: %s", err)
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
		participantChannel := not.centrifuge.PersonalChannel(utils.Int64ToString(participantId))
		GetLogEntry(c.Request()).Infof("Sending notification about create the chat to participantChannel: %v", participantChannel)

		unreadMessages, err := tx.GetUnreadMessages(chatId, participantId)
		if err != nil {
			GetLogEntry(c.Request()).Errorf("error during get unread messages for userId=%v: %s", participantId, err)
			continue
		}

		payload := &ChatUnreadMessageChanged{
			ChatId:         chatId,
			UnreadMessages: unreadMessages,
		}

		notification := CentrifugeNotification{
			Payload:   payload,
			EventType: "unread_messages_changed",
		}
		if marshalledBytes, err2 := json.Marshal(notification); err2 != nil {
			GetLogEntry(c.Request()).Errorf("error during marshalling chat created notification: %s", err2)
		} else {
			_, err := not.centrifuge.Publish(participantChannel, marshalledBytes)
			if err != nil {
				GetLogEntry(c.Request()).Errorf("error publishing to personal channel: %s", err)
			}
		}
	}
}

func messageNotifyCommon(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto, not *notifictionsImpl, eventType string) {
	// we send a notification only to those people who are currently reading the chat
	// if this is not done - when the user has many chats, he will receive many notifications and filter them on js
	activeChatUsers := []int64{}
	chatChannel := fmt.Sprintf("%v%v", utils.CHANNEL_PREFIX_CHAT_MESSAGES, chatId)
	presence, err := not.centrifuge.Presence(chatChannel)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("error during get chat presence for participantId : %s", err)
		return
	}
	for _, ci := range presence {
		if parseInt64, err := utils.ParseInt64(ci.User); err != nil {
			GetLogEntry(c.Request()).Errorf("error during parse participantId : %s", err)
		} else {
			activeChatUsers = append(activeChatUsers, parseInt64)
		}
	}

	for _, participantId := range userIds {
		if utils.Contains(activeChatUsers, participantId) {

			participantChannel := not.centrifuge.PersonalChannel(utils.Int64ToString(participantId))
			GetLogEntry(c.Request()).Infof("Sending notification about create the chat to participantChannel: %v", participantChannel)

			var copied *dto.DisplayMessageDto = &dto.DisplayMessageDto{}
			if err := deepcopy.Copy(copied, message); err != nil {
				GetLogEntry(c.Request()).Errorf("error during performing deep copy: %s", err)
				continue
			}

			// TODO move to better place
			copied.CanEdit = message.OwnerId == participantId

			dn := &DisplayMessageDtoNotification{
				*copied,
				chatId,
			}
			notification := CentrifugeNotification{
				Payload:   dn,
				EventType: eventType,
			}
			if marshalledBytes, err2 := json.Marshal(notification); err2 != nil {
				GetLogEntry(c.Request()).Errorf("error during marshalling chat created notification: %s", err2)
			} else {
				_, err := not.centrifuge.Publish(participantChannel, marshalledBytes)
				if err != nil {
					GetLogEntry(c.Request()).Errorf("error publishing to personal channel: %s", err)
				}
			}
		} else {
			GetLogEntry(c.Request()).Warnf("User %v is not present in chat %v, skipping notification", participantId, chatId)
		}
	}
}

func (not *notifictionsImpl) NotifyAboutNewMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_created")
}

func (not *notifictionsImpl) NotifyAboutDeleteMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_deleted")
}

func (not *notifictionsImpl) NotifyAboutEditMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto) {
	messageNotifyCommon(c, userIds, chatId, message, not, "message_edited")
}

func (not *notifictionsImpl) NotifyAboutMessageTyping(c echo.Context, chatId int64, user *dto.User) {
	if user == nil {
		GetLogEntry(c.Request()).Errorf("user cannot be null")
		return
	}

	channelName := fmt.Sprintf("%v%v", utils.CHANNEL_PREFIX_CHAT_MESSAGES, chatId)

	ut := UserTypingNotification{
		Login:         user.Login,
		ParticipantId: user.Id,
	}

	notification := CentrifugeNotification{
		Payload:   ut,
		EventType: "user_typing",
	}

	if marshalledBytes, err2 := json.Marshal(notification); err2 != nil {
		GetLogEntry(c.Request()).Errorf("error during marshalling chat created UserTypingNotification: %s", err2)
	} else {
		_, err := not.centrifuge.Publish(channelName, marshalledBytes)
		if err != nil {
			GetLogEntry(c.Request()).Errorf("error publishing to personal channel: %s", err)
		}
	}

}
