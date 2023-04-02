package dto

type MentionNotification struct {
	Id   int64  `json:"id"`
	Text string `json:"text"`
}

type MissedCallNotification struct {
	Description string `json:"description"`
}

type ReplyDto struct {
	MessageId        int64  `json:"messageId"`
	ChatId           int64  `json:"chatId"`
	ReplyableMessage string `json:"replyableMessage"`
}

type NotificationEvent struct {
	EventType              string                  `json:"eventType"`
	ChatId                 int64                   `json:"chatId"`
	UserId                 int64                   `json:"userId"`
	MentionNotification    *MentionNotification    `json:"mentionNotification"`
	MissedCallNotification *MissedCallNotification `json:"missedCallNotification"`
	ReplyNotification      *ReplyDto               `json:"replyNotification"`
	ByUserId               int64                   `json:"byUserId"`
	ByLogin                string                  `json:"byLogin"`
}

type GlobalEvent struct {
	EventType             string           `json:"eventType"`
	UserId                int64            `json:"userId"`
	UserNotificationEvent *NotificationDto `json:"userNotificationEvent"`
}
