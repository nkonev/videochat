package dto

type MentionNotification struct {
	Id   int64  `json:"id"`
	Text string `json:"text"`
}

type ReactionEvent struct {
	UserId    int64  `json:"userId"` // who gave this reaction
	Reaction  string `json:"reaction"`
	MessageId int64  `json:"messageId"`
}

type ReplyDto struct {
	MessageId        int64  `json:"messageId"`
	ChatId           int64  `json:"chatId"`
	ReplyableMessage string `json:"replyableMessage"`
}

type NotificationEvent struct {
	EventType           string               `json:"eventType"`
	ChatId              int64                `json:"chatId"`
	UserId              int64                `json:"userId"`
	ByUserId            int64                `json:"byUserId"`
	ByLogin             string               `json:"byLogin"`
	ByAvatar            *string              `json:"byAvatar"`
	ChatTitle           string               `json:"chatTitle"`
	MentionNotification *MentionNotification `json:"mentionNotification"`
	ReplyNotification   *ReplyDto            `json:"replyNotification"`
	ReactionEvent       *ReactionEvent       `json:"reactionEvent"`
}
