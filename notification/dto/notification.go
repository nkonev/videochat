package dto

import (
	"time"
)

type NotificationDto struct {
	Id               int64     `json:"id"`
	ChatId           int64     `json:"chatId"`
	MessageId        *int64    `json:"messageId"`
	NotificationType string    `json:"notificationType"`
	Description      string    `json:"description"`
	CreateDateTime   time.Time `json:"createDateTime"`
}

type NotificationSettings struct {
	MentionsEnabled    bool `json:"mentionsEnabled"`
	MissedCallsEnabled bool `json:"missedCallsEnabled"`
}

func NewNotificationDeleteDto(id int64) *NotificationDto {
	return &NotificationDto{
		Id:             id,
		CreateDateTime: time.Now(), // it needs for GraphLQ because this field is not nullable
	}
}
