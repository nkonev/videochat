package dto

import "github.com/montag451/go-eventbus"

const MESSAGE_NOTIFY_COMMON = "message.notify.common"

type MessageNotify struct {
	Type                string
	MessageNotification *DisplayMessageDto
}

func (MessageNotify) Name() eventbus.EventName {
	return MESSAGE_NOTIFY_COMMON
}
