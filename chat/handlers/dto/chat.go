package dto

import (
	"github.com/guregu/null"
	"time"
)

type ChatDto struct {
	Id                 int64     `json:"id"`
	Name               string    `json:"name"`
	LastUpdateDateTime time.Time `json:"lastUpdateDateTime"`
	ParticipantIds     []int64   `json:"participantIds"`
	Participants       []*User   `json:"participants"`
	CanEdit            null.Bool `json:"canEdit"`
	CanDelete            null.Bool `json:"canDelete"`
	CanLeave           null.Bool `json:"canLeave"`
	UnreadMessages     int64     `json:"unreadMessages"`
	CanBroadcast 	   bool 	 `json:"canBroadcast"`
	CanVideoKick	   bool 	 `json:"canVideoKick"`
	CanChangeChatAdmins	   bool 	 `json:"canChangeChatAdmins"`
	IsTetATet			   bool 	 `json:"tetATet"`
}

type ChatDtoWithTetATet interface {
	GetId() int64
	GetName() string
	GetIsTetATet() bool
	SetName(s string)
}

func (r *ChatDto) GetId() int64 {
	return r.Id
}

func (r *ChatDto) GetName() string {
	return r.Name
}

func (r *ChatDto) SetName(s string) {
	r.Name = s
}

func (r *ChatDto) GetIsTetATet() bool {
	return r.IsTetATet
}

// copied view for GET /chat/:id
type ChatDtoWithAdmin struct {
	Id                 int64     `json:"id"`
	Name               string    `json:"name"`
	LastUpdateDateTime time.Time `json:"lastUpdateDateTime"`
	ParticipantIds     []int64   `json:"participantIds"`
	Participants       []*UserWithAdmin   `json:"participants"`
	CanEdit            null.Bool `json:"canEdit"`
	CanDelete            null.Bool `json:"canDelete"`
	CanLeave           null.Bool `json:"canLeave"`
	UnreadMessages     int64     `json:"unreadMessages"`
	CanBroadcast 	   bool 	 `json:"canBroadcast"`
	CanVideoKick	   bool 	 `json:"canVideoKick"`
	CanChangeChatAdmins	   bool 	 `json:"canChangeChatAdmins"`
	IsTetATet			   bool 	 `json:"tetATet"`
}
