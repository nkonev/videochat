package dto

import (
	"github.com/guregu/null"
	"time"
)

const EmbedMessageTypeResend = "resend"
const EmbedMessageTypeReply = "reply"

type EmbedMessageResponse struct {
	Id            int64   `json:"id"`
	ChatId        *int64  `json:"chatId"`
	ChatName      *string `json:"chatName"`
	Text          string  `json:"text"`
	Owner         *User   `json:"owner"`
	EmbedType     string  `json:"embedType"`
	IsParticipant bool    `json:"isParticipant"`
}

type EmbedMessageRequest struct {
	Id        int64  `json:"id"`
	ChatId    int64  `json:"chatId"`
	EmbedType string `json:"embedType"`
}

type Reaction struct {
	Count    int64   `json:"count"`
	Users    []*User `json:"users"`
	Reaction string  `json:"reaction"`
}

type DisplayMessageDto struct {
	Id             int64                 `json:"id"`
	Text           string                `json:"text"`
	ChatId         int64                 `json:"chatId"`
	OwnerId        int64                 `json:"ownerId"`
	CreateDateTime time.Time             `json:"createDateTime"`
	EditDateTime   null.Time             `json:"editDateTime"`
	Owner          *User                 `json:"owner"`
	CanEdit        bool                  `json:"canEdit"`
	CanDelete      bool                  `json:"canDelete"`
	FileItemUuid   *string               `json:"fileItemUuid"`
	EmbedMessage   *EmbedMessageResponse `json:"embedMessage"`
	Pinned         bool                  `json:"pinned"`
	BlogPost       bool                  `json:"blogPost"`
	PinnedPromoted *bool                 `json:"pinnedPromoted"`
	Reactions      []Reaction            `json:"reactions"`
	Published      bool                  `json:"published"`
	CanPublish     bool                  `json:"canPublish"`
	CanPin         bool                  `json:"canPin"`
}

type PublishedMessageDto struct {
	Id             int64     `json:"id"`
	Text           string    `json:"text"`
	ChatId         int64     `json:"chatId"`
	OwnerId        int64     `json:"ownerId"`
	Owner          *User     `json:"owner"`
	CanPublish     bool      `json:"canPublish"`
	CreateDateTime time.Time `json:"createDateTime"`
}

type PinnedMessageDto struct {
	Id             int64     `json:"id"`
	Text           string    `json:"text"`
	ChatId         int64     `json:"chatId"`
	OwnerId        int64     `json:"ownerId"`
	Owner          *User     `json:"owner"`
	PinnedPromoted bool      `json:"pinnedPromoted"`
	CreateDateTime time.Time `json:"createDateTime"`
	CanPin         bool      `json:"canPin"`
}

func CanPublishMessage(chatRegularParticipantCanPublishMessage, chatIsAdmin bool, messageOwnerId, behalfUserId int64) bool {
	return chatIsAdmin || (chatRegularParticipantCanPublishMessage && messageOwnerId == behalfUserId)
}

func CanPinMessage(chatRegularParticipantCanPinMessage, chatIsAdmin bool) bool {
	return chatIsAdmin || chatRegularParticipantCanPinMessage
}

func (copied *DisplayMessageDto) SetPersonalizedFields(chatRegularParticipantCanPublishMessage, chatRegularParticipantCanPinMessage, chatCanWriteMessage, chatIsAdmin bool, participantId int64) {
	canWriteMessage := chatIsAdmin || chatCanWriteMessage

	copied.CanEdit = ((copied.OwnerId == participantId) && (copied.EmbedMessage == nil || copied.EmbedMessage.EmbedType != EmbedMessageTypeResend)) && canWriteMessage
	copied.CanDelete = copied.OwnerId == participantId && canWriteMessage
	copied.CanPublish = CanPublishMessage(chatRegularParticipantCanPublishMessage, chatIsAdmin, copied.OwnerId, participantId)
	copied.CanPin = CanPinMessage(chatRegularParticipantCanPinMessage, chatIsAdmin)
}

type MessageDeletedDto struct {
	Id     int64 `json:"id"`
	ChatId int64 `json:"chatId"`
}

type UserTypingNotification struct {
	Login         string `json:"login"`
	ParticipantId int64  `json:"participantId"`
	ChatId        int64  `json:"chatId"`
}

type MessageBroadcastNotification struct {
	Login  string `json:"login"`
	UserId int64  `json:"userId"`
	Text   string `json:"text"`
}

type AllUnreadMessages struct {
	MessagesCount int64 `json:"allUnreadMessages"`
}

type ReplyDto struct {
	MessageId        int64  `json:"messageId"`
	ChatId           int64  `json:"chatId"`
	ReplyableMessage string `json:"replyableMessage"`
}
