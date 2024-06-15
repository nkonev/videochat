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
	ByUserId         int64     `json:"byUserId"`
	ByLogin          string    `json:"byLogin"`
	ByAvatar         *string   `json:"byAvatar"`
	ChatTitle        string    `json:"chatTitle"`
}

type WrapperNotificationDto struct {
	NotificationDto NotificationDto   `json:"notificationDto"`
	TotalCount      int64             `json:"totalCount"`
}

type NotificationGlobalSettings struct {
	MentionsEnabled    bool `json:"mentionsEnabled"`
	MissedCallsEnabled bool `json:"missedCallsEnabled"`
	AnswersEnabled     bool `json:"answersEnabled"`
	ReactionsEnabled   bool `json:"reactionsEnabled"`
}

type NotificationPerChatSettings struct {
	MentionsEnabled    *bool `json:"mentionsEnabled"`
	MissedCallsEnabled *bool `json:"missedCallsEnabled"`
	AnswersEnabled     *bool `json:"answersEnabled"`
	ReactionsEnabled   *bool `json:"reactionsEnabled"`
}

func NewNotificationDeleteDto(id int64, notificationType string) *NotificationDto {
	return &NotificationDto{
		Id:             id,
		CreateDateTime: time.Now(), // it needs for GraphQL because this field is not nullable
		NotificationType: notificationType,
	}
}

func NewWrapperNotificationDeleteDto(id int64, totalCount int64, notificationType string) *WrapperNotificationDto {
	tmp := NewNotificationDeleteDto(id, notificationType)
	return &WrapperNotificationDto{
		NotificationDto: *tmp,
		TotalCount: totalCount,
	}
}
