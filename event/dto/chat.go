package dto

import (
	"github.com/guregu/null"
	"time"
)

type BaseChatDto struct {
	Id                  int64       `json:"id"`
	Name                string      `json:"name"`
	Avatar              null.String `json:"avatar"`
	AvatarBig           null.String `json:"avatarBig"`
	ShortInfo           null.String `json:"shortInfo"`
	LastUpdateDateTime  time.Time   `json:"lastUpdateDateTime"`
	ParticipantIds      []int64     `json:"participantIds"`
	CanEdit             null.Bool   `json:"canEdit"`
	CanDelete           null.Bool   `json:"canDelete"`
	CanLeave            null.Bool   `json:"canLeave"`
	UnreadMessages      int64       `json:"unreadMessages"`
	CanBroadcast        bool        `json:"canBroadcast"`
	CanVideoKick        bool        `json:"canVideoKick"`
	CanChangeChatAdmins bool        `json:"canChangeChatAdmins"`
	IsTetATet           bool        `json:"tetATet"`
	CanAudioMute        bool        `json:"canAudioMute"`
	ParticipantsCount   int         `json:"participantsCount"`
	CanResend           bool        `json:"canResend"`
	AvailableToSearch   bool        `json:"availableToSearch"`
	IsResultFromSearch  null.Bool   `json:"isResultFromSearch"`
	Pinned              bool        `json:"pinned"`
	Blog                bool        `json:"blog"`
}

type ChatDeletedDto struct {
	Id int64 `json:"id"`
}

type ChatDto struct {
	BaseChatDto
	Participants []*User `json:"participants"`
}

// copied view for GET /chat/:id
type ChatDtoWithAdmin struct {
	BaseChatDto
	Participants []*UserWithAdmin `json:"participants"`
}

type ChatUnreadMessageChanged struct {
	ChatId             int64     `json:"chatId"`
	UnreadMessages     int64     `json:"unreadMessages"`
	LastUpdateDateTime time.Time `json:"lastUpdateDateTime"`
}
