package dto

import "github.com/montag451/go-eventbus"

const CHAT_EVENTS = "events.chat"
const GLOBAL_EVENTS = "events.global"

type ChatEvent struct {
	EventType           string
	UserIds             *[]int64
	MessageNotification *DisplayMessageDto
}

func (ChatEvent) Name() eventbus.EventName {
	return CHAT_EVENTS
}

type GlobalEvent struct {
	EventType        string
	UserIds          *[]int64
	ChatNotification *ChatDtoWithAdmin
}

func (GlobalEvent) Name() eventbus.EventName {
	return GLOBAL_EVENTS
}
