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
	CanLeave           null.Bool `json:"canLeave"`
	UnreadMessages     int64     `json:"unreadMessages"`
	CanBroadcast 	   bool 	 `json:"canBroadcast"`
	CanVideoKick	   bool 	 `json:"canVideoKick"`
	CanChangeChatAdmins	   bool 	 `json:"canChangeChatAdmins"`
	IsTetATet			   bool 	 `json:"tetATet"`
}

type ChatDtoWithAdmin struct {
	Id                 int64     `json:"id"`
	Name               string    `json:"name"`
	LastUpdateDateTime time.Time `json:"lastUpdateDateTime"`
	ParticipantIds     []int64   `json:"participantIds"`
	Participants       []*UserWithAdmin   `json:"participants"`
	CanEdit            null.Bool `json:"canEdit"`
	CanLeave           null.Bool `json:"canLeave"`
	UnreadMessages     int64     `json:"unreadMessages"`
	CanBroadcast 	   bool 	 `json:"canBroadcast"`
	CanVideoKick	   bool 	 `json:"canVideoKick"`
	CanChangeChatAdmins	   bool 	 `json:"canChangeChatAdmins"`
	IsTetATet			   bool 	 `json:"tetATet"`
}
