package dto

import "github.com/montag451/go-eventbus"

const NOTIFY_COMMON = "notify.common"

type EventBusEvent struct {
	EventType           string
	MessageNotification *DisplayMessageDto
	ChatNotification    *ChatDtoWithAdmin
}

func (EventBusEvent) Name() eventbus.EventName {
	return NOTIFY_COMMON
}
