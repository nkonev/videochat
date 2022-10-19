package dto

import "github.com/montag451/go-eventbus"

const CHAT_EVENTS = "events.chat"
const GLOBAL_EVENTS = "events.global"

type ChatEvent struct {
	EventType           string             `json:"eventType"`
	ChatId              int64              `json:"chatId"`
	UserId              int64              `json:"userId"`
	MessageNotification *DisplayMessageDto `json:"messageNotification"`
}

func (ChatEvent) Name() eventbus.EventName {
	return CHAT_EVENTS
}

type GlobalEvent struct {
	EventType        string            `json:"eventType"`
	UserId           int64             `json:"userId"`
	ChatNotification *ChatDtoWithAdmin `json:"chatNotification"`
}

func (GlobalEvent) Name() eventbus.EventName {
	return GLOBAL_EVENTS
}
