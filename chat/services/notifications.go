package services

import (
	"github.com/centrifugal/centrifuge"
	"github.com/getlantern/deepcopy"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/utils"
)

type Notifications interface {
	NotifyAboutNewChat(c echo.Context, newChatDto *dto.ChatDtoWithAdmin, userIds []int64, tx *db.Tx)
	NotifyAboutDeleteChat(c echo.Context, chatId int64, userIds []int64, tx *db.Tx)
	NotifyAboutChangeChat(c echo.Context, chatDto *dto.ChatDtoWithAdmin, userIds []int64, changingParticipantPage int, tx *db.Tx)
	NotifyAboutNewMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto)
	NotifyAboutDeleteMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto)
	NotifyAboutEditMessage(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto)
	NotifyAboutProfileChanged(user *dto.User)
	NotifyAboutMessageTyping(c echo.Context, chatId int64, user *dto.User)
	NotifyAboutMessageBroadcast(c echo.Context, chatId, userId int64, login, text string)
	ChatNotifyMessageCount(userIds []int64, c echo.Context, chatId int64, tx *db.Tx)
	ChatNotifyAllUnreadMessageCount(userIds []int64, c echo.Context, tx *db.Tx)
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
	dto.DisplayMessageDto
	ChatId int64 `json:"chatId"`
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

func (not *notifictionsImpl) NotifyAboutNewChat(c echo.Context, newChatDto *dto.ChatDtoWithAdmin, userIds []int64, tx *db.Tx) {
	chatNotifyCommon(userIds, not, c, newChatDto, "chat_created", NoPagePlaceholder, tx)
}

func (not *notifictionsImpl) NotifyAboutChangeChat(c echo.Context, chatDto *dto.ChatDtoWithAdmin, userIds []int64, changingParticipantPage int, tx *db.Tx) {
	chatNotifyCommon(userIds, not, c, chatDto, "chat_edited", changingParticipantPage, tx)
}

func (not *notifictionsImpl) NotifyAboutDeleteChat(c echo.Context, chatId int64, userIds []int64, tx *db.Tx) {
	chatDto := dto.ChatDtoWithAdmin{
		BaseChatDto: dto.BaseChatDto{
			Id: chatId,
		},
	}
	chatNotifyCommon(userIds, not, c, &chatDto, "chat_deleted", NoPagePlaceholder, tx)
}

func chatNotifyCommon(userIds []int64, not *notifictionsImpl, c echo.Context, newChatDto *dto.ChatDtoWithAdmin, eventType string, changingParticipantPage int, tx *db.Tx) {
	for _, participantId := range userIds {
		GetLogEntry(c.Request().Context()).Infof("Sending notification about %v the chat to participant: %v", eventType, participantId)

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

		// TODO move to better place
		//  see also handlers/chat.go:199 convertToDto()
		copied.CanEdit = null.BoolFrom(admin && !copied.IsTetATet)
		copied.CanDelete = null.BoolFrom(admin)
		copied.CanLeave = null.BoolFrom(!admin && !copied.IsTetATet)
		copied.UnreadMessages = unreadMessages
		copied.CanVideoKick = admin
		copied.CanAudioMute = admin
		copied.CanChangeChatAdmins = admin && !copied.IsTetATet
		copied.ParticipantsCount = newChatDto.ParticipantsCount
		copied.ChangingParticipantsPage = changingParticipantPage
		copied.CanBroadcast = admin
		for _, participant := range copied.Participants {
			utils.ReplaceChatNameToLoginForTetATet(copied, &participant.User, participantId)
		}

		err = not.rabbitPublisher.Publish(dto.GlobalEvent{
			UserId:           participantId,
			EventType:        eventType,
			ChatNotification: newChatDto,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *notifictionsImpl) ChatNotifyMessageCount(userIds []int64, c echo.Context, chatId int64, tx *db.Tx) {
	for _, participantId := range userIds {
		GetLogEntry(c.Request().Context()).Infof("Sending notification about unread messages to participantChannel: %v", participantId)

		unreadMessages, err := tx.GetUnreadMessagesCount(chatId, participantId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during get unread messages for userId=%v: %s", participantId, err)
			continue
		}

		payload := &dto.ChatUnreadMessageChanged{
			ChatId:         chatId,
			UnreadMessages: unreadMessages,
		}

		err = not.rabbitPublisher.Publish(dto.GlobalEvent{
			UserId:                     participantId,
			EventType:                  "chat_unread_messages_changed",
			UnreadMessagesNotification: payload,
		})
	}
}

func (not *notifictionsImpl) ChatNotifyAllUnreadMessageCount(userIds []int64, c echo.Context, tx *db.Tx) {
	for _, participantId := range userIds {
		GetLogEntry(c.Request().Context()).Infof("Sending notification about all unread messages to participantChannel: %v", participantId)

		unreadMessages, err := tx.GetAllUnreadMessagesCount(participantId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during get all unread messages for userId=%v: %s", participantId, err)
			continue
		}

		payload := &dto.AllUnreadMessages{
			MessagesCount: unreadMessages,
		}

		err = not.rabbitPublisher.Publish(dto.GlobalEvent{
			UserId:                        participantId,
			EventType:                     "all_unread_messages_changed",
			AllUnreadMessagesNotification: payload,
		})
	}
}

func messageNotifyCommon(c echo.Context, userIds []int64, chatId int64, message *dto.DisplayMessageDto, not *notifictionsImpl, eventType string) {
	for _, participantId := range userIds {
		err := not.rabbitPublisher.Publish(dto.ChatEvent{
			EventType:           eventType,
			MessageNotification: message,
			UserId:              participantId,
			ChatId:              chatId,
		})
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during sending to rabbitmq : %s", err)
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
		err := not.rabbitPublisher.Publish(dto.ChatEvent{
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

func (not *notifictionsImpl) NotifyAboutProfileChanged(user *dto.User) {
	if user == nil {
		Logger.Errorf("user cannot be null")
		return
	}

	coChatters, err := not.db.GetCoChattedParticipantIdsCommon(user.Id)
	if err != nil {
		Logger.Errorf("Error during get co-chatters for %v, error: %v", user.Id, err)
	}

	for _, participantId := range coChatters {
		err = not.rabbitPublisher.Publish(dto.GlobalEvent{
			UserId:                  participantId,
			EventType:               "user_profile_changed",
			UserProfileNotification: user,
		})
		if err != nil {
			Logger.Errorf("Error during sending to rabbitmq : %s", err)
		}
	}
}

func (not *notifictionsImpl) NotifyAboutMessageBroadcast(c echo.Context, chatId, userId int64, login, text string) {
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
		err := not.rabbitPublisher.Publish(dto.ChatEvent{
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
