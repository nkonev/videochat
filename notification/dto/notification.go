package dto

import "time"

type NotificationDto struct {
	Id               int64     `json:"id"`
	ChatId           int64     `json:"chatId"`
	MessageId        *int64    `json:"messageId"`
	NotificationType string    `json:"notificationType"`
	Description      string    `json:"description"`
	CreateDateTime   time.Time `json:"createDateTime"`
}

type NotificationEventDto struct {
	Dto       NotificationDto `json:"dto"`
	EventType string          `json:"eventType"`
}

type NotificationSettings struct {
	MentionsEnabled    bool `json:"mentionsEnabled"`
	MissedCallsEnabled bool `json:"missedCallsEnabled"`
}
