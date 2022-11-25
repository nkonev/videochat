package dto

import "time"

type NotificationDto struct {
	Id             int64     `json:"id"`
	ChatId         int64     `json:"chatId"`
	MessageId      *int64    `json:"messageId"`
	Type           string    `json:"type"`
	Description    string    `json:"description"`
	CreateDateTime time.Time `json:"createDateTime"`
}
