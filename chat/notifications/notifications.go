package notifications

import (
	"encoding/json"
	"fmt"
	"github.com/centrifugal/centrifuge"
	"github.com/getlantern/deepcopy"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

type Notifications interface {
	NotifyAboutNewChat(c echo.Context, newChatDto *dto.ChatDto, userIds []int64, tx *db.Tx)
	NotifyAboutDeleteChat(c echo.Context, chatDto *dto.ChatDto, userIds []int64, tx *db.Tx)
	NotifyAboutChangeChat(c echo.Context, chatDto *dto.ChatDto, userIds []int64, tx *db.Tx)
	NotifyAboutNewMessage(c echo.Context, chatId int64, message *dto.DisplayMessageDto, userPrincipalDto *auth.AuthResult)
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

func (not *notifictionsImpl) NotifyAboutNewChat(c echo.Context, newChatDto *dto.ChatDto, userIds []int64, tx *db.Tx) {
	notifyCommon(userIds, not, c, newChatDto, "chat_created", tx)
}

func (not *notifictionsImpl) NotifyAboutChangeChat(c echo.Context, chatDto *dto.ChatDto, userIds []int64, tx *db.Tx) {
	notifyCommon(userIds, not, c, chatDto, "chat_edited", tx)
}

func (not *notifictionsImpl) NotifyAboutDeleteChat(c echo.Context, chatDto *dto.ChatDto, userIds []int64, tx *db.Tx) {
	notifyCommon(userIds, not, c, chatDto, "chat_deleted", tx)
}

func notifyCommon(userIds []int64, not *notifictionsImpl, c echo.Context, newChatDto *dto.ChatDto, eventType string, tx *db.Tx) {
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

		// TODO rework or merge
		copied.CanEdit = null.BoolFrom(admin)
		copied.CanLeave = null.BoolFrom(!admin)

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

func (not *notifictionsImpl) NotifyAboutNewMessage(c echo.Context, chatId int64, message *dto.DisplayMessageDto, userPrincipalDto *auth.AuthResult) {
	chatChannel := fmt.Sprintf("%v%v", utils.CHANNEL_PREFIX_CHAT, chatId)
	notification := CentrifugeNotification{
		Payload:   *message,
		EventType: "message_created",
	}
	if marshalledBytes, err2 := json.Marshal(notification); err2 != nil {
		GetLogEntry(c.Request()).Errorf("error during marshalling chat created notification: %s", err2)
	} else {
		_, err := not.centrifuge.Publish(chatChannel, marshalledBytes)
		if err != nil {
			GetLogEntry(c.Request()).Errorf("error publishing to personal channel: %s", err)
		}
	}
}
