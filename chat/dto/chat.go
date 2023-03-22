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
}

func (copied *BaseChatDto) SetPersonalizedFields(admin bool, unreadMessages int64) {
	copied.CanEdit = null.BoolFrom(admin && !copied.IsTetATet)
	copied.CanDelete = null.BoolFrom(admin)
	copied.CanLeave = null.BoolFrom(!admin && !copied.IsTetATet)
	copied.UnreadMessages = unreadMessages
	copied.CanVideoKick = admin
	copied.CanAudioMute = admin
	copied.CanChangeChatAdmins = admin && !copied.IsTetATet
	copied.CanBroadcast = admin
}

type ChatDeletedDto struct {
	Id int64 `json:"id"`
}

type ChatDto struct {
	BaseChatDto
	Participants []*User `json:"participants"`
}

type ChatDtoWithTetATet interface {
	GetId() int64
	GetName() string
	GetAvatar() null.String
	GetIsTetATet() bool
	SetName(s string)
	SetAvatar(s null.String)
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

func (r *BaseChatDto) GetId() int64 {
	return r.Id
}

func (r *BaseChatDto) GetName() string {
	return r.Name
}

func (r *BaseChatDto) SetName(s string) {
	r.Name = s
}

func (r *BaseChatDto) GetAvatar() null.String {
	return r.Avatar
}

func (r *BaseChatDto) SetAvatar(s null.String) {
	r.Avatar = s
}

func (r *BaseChatDto) GetIsTetATet() bool {
	return r.IsTetATet
}

// copied view for GET /chat/:id
type ChatDtoWithAdmin struct {
	BaseChatDto
	Participants []*UserWithAdmin `json:"participants"`
}

type ChatName struct {
	Name   string      `json:"name"`   // chatName or userName in case tet-a-tet
	Avatar null.String `json:"avatar"` // tet-a-tet -aware
	UserId int64       `json:"userId"` // userId chatName for
}

type ChatUnreadMessageChanged struct {
	ChatId         int64 `json:"chatId"`
	UnreadMessages int64 `json:"unreadMessages"`
}

type BasicChatDto struct {
	TetATet        bool    `json:"tetATet"`
	ParticipantIds []int64 `json:"participantIds"`
}
