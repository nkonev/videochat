package dto

import "github.com/montag451/go-eventbus"

const MESSAGE_NOTIFY_COMMON = "message.notify.common"

type ChatEvent struct {
	EventType           string
	MessageNotification *DisplayMessageDto
}

func (ChatEvent) Name() eventbus.EventName {
	return MESSAGE_NOTIFY_COMMON
}
