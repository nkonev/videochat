package dto

import (
	"time"
)

type BaseChatDto struct {
	Id                                  int64           `json:"id"`
	Name                                string          `json:"name"`
	Avatar                              *string         `json:"avatar"`
	AvatarBig                           *string         `json:"avatarBig"`
	ShortInfo                           *string         `json:"shortInfo"`
	LastUpdateDateTime                  time.Time       `json:"lastUpdateDateTime"`
	ParticipantIds                      []int64         `json:"participantIds"`
	CanEdit                             *bool           `json:"canEdit"`
	CanDelete                           *bool           `json:"canDelete"`
	CanLeave                            *bool           `json:"canLeave"`
	UnreadMessages                      int64           `json:"unreadMessages"`
	CanBroadcast                        bool            `json:"canBroadcast"`
	CanVideoKick                        bool            `json:"canVideoKick"`
	CanChangeChatAdmins                 bool            `json:"canChangeChatAdmins"`
	IsTetATet                           bool            `json:"tetATet"`
	CanAudioMute                        bool            `json:"canAudioMute"`
	ParticipantsCount                   int             `json:"participantsCount"`
	CanResend                           bool            `json:"canResend"`
	AvailableToSearch                   bool            `json:"availableToSearch"`
	IsResultFromSearch                  *bool           `json:"isResultFromSearch"`
	Pinned                              bool            `json:"pinned"`
	Blog                                bool            `json:"blog"`
	LoginColor                          *string         `json:"loginColor"`
	RegularParticipantCanPublishMessage bool            `json:"regularParticipantCanPublishMessage"`
	LastSeenDateTime                    *time.Time      `json:"lastSeenDateTime"`
	RegularParticipantCanPinMessage     bool            `json:"regularParticipantCanPinMessage"`
	BlogAbout                           bool            `json:"blogAbout"`
	RegularParticipantCanWriteMessage   bool            `json:"regularParticipantCanWriteMessage"`
	CanWriteMessage                     bool            `json:"canWriteMessage"`
	CanReact                            bool            `json:"canReact"`
	AdditionalData                      *AdditionalData `json:"additionalData"`
}

func (copied *BaseChatDto) SetPersonalizedFields(admin bool, unreadMessages int64, participant bool, pinned bool) {
	canEdit := admin && !copied.IsTetATet
	copied.CanEdit = &canEdit
	canDelete := admin
	copied.CanDelete = &canDelete
	canLeave := !admin && !copied.IsTetATet && participant
	copied.CanLeave = &canLeave
	copied.UnreadMessages = unreadMessages
	copied.CanVideoKick = admin
	copied.CanAudioMute = admin
	copied.CanChangeChatAdmins = admin && !copied.IsTetATet
	copied.CanBroadcast = admin

	if !participant {
		isResultFromSearch := true
		copied.IsResultFromSearch = &isResultFromSearch
	}

	copied.CanWriteMessage = true
	// see also handlers PostMessage, EditMessage, DeleteMessage
	if !copied.RegularParticipantCanWriteMessage && !admin {
		copied.CanWriteMessage = false
	}

	copied.Pinned = pinned
}

type ChatDeletedDto struct {
	Id int64 `json:"id"`
}

type ChatDto struct {
	BaseChatDto
	Participants       []*User `json:"participants"`
	LastMessagePreview *string `json:"lastMessagePreview"`
}

type ChatDtoWithTetATet interface {
	GetId() int64
	GetName() string
	GetAvatar() *string
	GetIsTetATet() bool
	SetName(s string)
	SetAvatar(s *string)
	SetShortInfo(s *string)
	SetLoginColor(s *string)
	SetLastSeenDateTime(t *time.Time)
	SetAdditionalData(ad *AdditionalData)
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

func (r *BaseChatDto) GetAvatar() *string {
	return r.Avatar
}

func (r *BaseChatDto) SetAvatar(s *string) {
	r.Avatar = s
}

func (r *BaseChatDto) SetShortInfo(s *string) {
	r.ShortInfo = s
}

func (r *BaseChatDto) SetLoginColor(s *string) {
	r.LoginColor = s
}

func (r *BaseChatDto) GetIsTetATet() bool {
	return r.IsTetATet
}

func (r *BaseChatDto) SetLastSeenDateTime(t *time.Time) {
	r.LastSeenDateTime = t
}

func (r *BaseChatDto) SetAdditionalData(ad *AdditionalData) {
	r.AdditionalData = ad
}

type ChatName struct {
	Name   string  `json:"name"`   // chatName or userName in case tet-a-tet
	Avatar *string `json:"avatar"` // tet-a-tet -aware
	UserId int64   `json:"userId"` // userId chatName for
}

type ChatUnreadMessageChanged struct {
	ChatId             int64     `json:"chatId"`
	UnreadMessages     int64     `json:"unreadMessages"`
	LastUpdateDateTime time.Time `json:"lastUpdateDateTime"`
}

type BasicChatDto struct {
	TetATet        bool    `json:"tetATet"`
	ParticipantIds []int64 `json:"participantIds"`
}
