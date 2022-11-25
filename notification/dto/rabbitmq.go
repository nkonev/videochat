package dto

type MentionNotification struct {
	Id   int64  `json:"id"`
	Text string `json:"text"`
}

type MissedCallNotification struct {
	Description string `json:"description"`
}

type NotificationEvent struct {
	EventType              string                  `json:"eventType"`
	ChatId                 int64                   `json:"chatId"`
	UserId                 int64                   `json:"userId"`
	MentionNotification    *MentionNotification    `json:"mentionNotification"`
	MissedCallNotification *MissedCallNotification `json:"missedCallNotification"`
}
