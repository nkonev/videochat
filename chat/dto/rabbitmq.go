package dto

type PinnedMessageEvent struct {
	Message    PinnedMessageDto `json:"message"`
	TotalCount int64            `json:"totalCount"`
}

type PublishedMessageEvent struct {
	Message    PublishedMessageDto `json:"message"`
	TotalCount int64               `json:"totalCount"`
}

type ReactionChangedEvent struct {
	MessageId int64    `json:"messageId"`
	Reaction  Reaction `json:"reaction"`
}

type ChatEvent struct {
	EventType                    string                        `json:"eventType"`
	ChatId                       int64                         `json:"chatId"`
	UserId                       int64                         `json:"userId"`
	MessageNotification          *DisplayMessageDto            `json:"messageNotification"`
	MessageDeletedNotification   *MessageDeletedDto            `json:"messageDeletedNotification"`
	MessageBroadcastNotification *MessageBroadcastNotification `json:"messageBroadcastNotification"`
	Participants                 *[]*UserWithAdmin             `json:"participants"`
	PromoteMessageNotification   *PinnedMessageEvent           `json:"promoteMessageNotification"`
	PublishedMessageNotification *PublishedMessageEvent        `json:"publishedMessageEvent"`
	ReactionChangedEvent         *ReactionChangedEvent         `json:"reactionChangedEvent"`
}

type HasUnreadMessagesChanged struct {
	HasUnreadMessages bool `json:"hasUnreadMessages"`
}

type BrowserNotification struct {
	ChatId      int64   `json:"chatId"`
	ChatName    string  `json:"chatName"`
	ChatAvatar  *string `json:"chatAvatar"`
	MessageId   int64   `json:"messageId"`
	MessageText string  `json:"messageText"`
	OwnerId     int64   `json:"ownerId"`
	OwnerLogin  string  `json:"ownerLogin"`
}

type GlobalUserEvent struct {
	EventType                        string                    `json:"eventType"`
	UserId                           int64                     `json:"userId"`
	ChatNotification                 *ChatDto                  `json:"chatNotification"`
	ChatDeletedDto                   *ChatDeletedDto           `json:"chatDeletedNotification"`
	CoChattedParticipantNotification *User                     `json:"coChattedParticipantNotification"`
	UnreadMessagesNotification       *ChatUnreadMessageChanged `json:"unreadMessagesNotification"`
	AllUnreadMessagesNotification    *AllUnreadMessages        `json:"allUnreadMessagesNotification"`
	HasUnreadMessagesChanged         *HasUnreadMessagesChanged `json:"hasUnreadMessagesChanged"`
	BrowserNotification              *BrowserNotification      `json:"browserNotification"`
	UserTypingNotification           *UserTypingNotification   `json:"userTypingNotification"`
}

type MentionNotification struct {
	Id   int64  `json:"id"`
	Text string `json:"text"`
}

type ReactionEvent struct {
	UserId    int64  `json:"userId"` // who gave this reaction
	Reaction  string `json:"reaction"`
	MessageId int64  `json:"messageId"`
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
