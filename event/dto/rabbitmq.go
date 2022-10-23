package dto

import "github.com/montag451/go-eventbus"

const CHAT_EVENTS = "events.chat"
const GLOBAL_EVENTS = "events.global"

type ChatEvent struct {
	EventType                    string                        `json:"eventType"`
	ChatId                       int64                         `json:"chatId"`
	UserId                       int64                         `json:"userId"`
	MessageNotification          *DisplayMessageDto            `json:"messageNotification"`
	UserTypingNotification       *UserTypingNotification       `json:"userTypingNotification"`
	MessageBroadcastNotification *MessageBroadcastNotification `json:"messageBroadcastNotification"`
}

func (ChatEvent) Name() eventbus.EventName {
	return CHAT_EVENTS
}

type GlobalEvent struct {
	EventType                  string                    `json:"eventType"`
	UserId                     int64                     `json:"userId"`
	ChatNotification           *ChatDtoWithAdmin         `json:"chatNotification"`
	UserProfileNotification    *User                     `json:"userProfileNotification"`
	VideoNotification          *VideoCallChangedDto      `json:"videoNotification"`
	VideoChatInvitation        *VideoCallInvitation      `json:"videoCallInvitation"`
	VideoParticipantDialEvent  *VideoDialChanges         `json:"videoParticipantDialEvent"`
	UnreadMessagesNotification *ChatUnreadMessageChanged `json:"unreadMessagesNotification"`
}

func (GlobalEvent) Name() eventbus.EventName {
	return GLOBAL_EVENTS
}
