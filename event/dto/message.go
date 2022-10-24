package dto

import (
	"github.com/google/uuid"
	"github.com/guregu/null"
	"time"
)

type DisplayMessageDto struct {
	Id             int64      `json:"id"`
	Text           string     `json:"text"`
	ChatId         int64      `json:"chatId"`
	OwnerId        int64      `json:"ownerId"`
	CreateDateTime time.Time  `json:"createDateTime"`
	EditDateTime   null.Time  `json:"editDateTime"`
	Owner          *User      `json:"owner"`
	CanEdit        bool       `json:"canEdit"`
	FileItemUuid   *uuid.UUID `json:"fileItemUuid"`
}

type UserTypingNotification struct {
	Login         string `json:"login"`
	ParticipantId int64  `json:"participantId"`
}

type MessageBroadcastNotification struct {
	Login  string `json:"login"`
	UserId int64  `json:"userId"`
	Text   string `json:"text"`
}

type AllUnreadMessages struct {
	MessagesCount int64 `json:"allUnreadMessages"`
}
