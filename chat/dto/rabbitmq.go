package dto

type ChatEvent struct {
	EventType                    string                        `json:"eventType"`
	ChatId                       int64                         `json:"chatId"`
	UserId                       int64                         `json:"userId"`
	MessageNotification          *DisplayMessageDto            `json:"messageNotification"`
	MessageDeletedNotification   *MessageDeletedDto            `json:"messageDeletedNotification"`
	UserTypingNotification       *UserTypingNotification       `json:"userTypingNotification"`
	MessageBroadcastNotification *MessageBroadcastNotification `json:"messageBroadcastNotification"`
	Participants                 *[]*UserWithAdmin             `json:"participants"`
	PromoteMessageNotification   *DisplayMessageDto            `json:"promoteMessageNotification"`
}

type GlobalEvent struct {
	EventType                     string                    `json:"eventType"`
	UserId                        int64                     `json:"userId"`
	ChatNotification              *ChatDtoWithAdmin         `json:"chatNotification"`
	ChatDeletedDto                *ChatDeletedDto           `json:"chatDeletedNotification"`
	UserProfileNotification       *User                     `json:"userProfileNotification"`
	UnreadMessagesNotification    *ChatUnreadMessageChanged `json:"unreadMessagesNotification"`
	AllUnreadMessagesNotification *AllUnreadMessages        `json:"allUnreadMessagesNotification"`
}

type MentionNotification struct {
	Id   int64  `json:"id"`
	Text string `json:"text"`
}

type NotificationEvent struct {
	EventType           string               `json:"eventType"`
	ChatId              int64                `json:"chatId"`
	UserId              int64                `json:"userId"`
	ByUserId            int64                `json:"byUserId"`
	ByLogin             string               `json:"byLogin"`
	MentionNotification *MentionNotification `json:"mentionNotification"`
	ReplyNotification   *ReplyDto            `json:"replyNotification"`
}
