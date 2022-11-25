package dto

type MentionNotification struct {
	Id   int64  `json:"id"`
	Text string `json:"text"`
}

type NotificationEvent struct {
	EventType              string               `json:"eventType"`
	ChatId                 int64                `json:"chatId"`
	UserId                 int64                `json:"userId"`
	MentionNotification    *MentionNotification `json:"mentionNotification"`
	MissedCallNotification bool                 `json:"missedCallNotification"`
}
