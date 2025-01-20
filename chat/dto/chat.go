package dto

import (
	"github.com/guregu/null"
	"time"
)

type BaseChatDto0 struct {
	Id                                  int64       `json:"id"`
	Name                                string      `json:"name"`
	Avatar                              null.String `json:"avatar"`
	AvatarBig                           null.String `json:"avatarBig"`
	ShortInfo                           null.String `json:"shortInfo"`
	LastUpdateDateTime                  time.Time   `json:"lastUpdateDateTime"`
	ParticipantIds                      []int64     `json:"participantIds"`
	CanEdit                             null.Bool   `json:"canEdit"`
	CanDelete                           null.Bool   `json:"canDelete"`
	CanLeave                            null.Bool   `json:"canLeave"`
	UnreadMessages                      int64       `json:"unreadMessages"`
	CanBroadcast                        bool        `json:"canBroadcast"`
	CanVideoKick                        bool        `json:"canVideoKick"`
	CanChangeChatAdmins                 bool        `json:"canChangeChatAdmins"`
	IsTetATet                           bool        `json:"tetATet"`
	CanAudioMute                        bool        `json:"canAudioMute"`
	ParticipantsCount                   int         `json:"participantsCount"`
	CanResend                           bool        `json:"canResend"`
	AvailableToSearch                   bool        `json:"availableToSearch"`
	IsResultFromSearch                  null.Bool   `json:"isResultFromSearch"`
	Pinned                              bool        `json:"pinned"`
	Blog                                bool        `json:"blog"`
	LoginColor                          null.String `json:"loginColor"`
	RegularParticipantCanPublishMessage bool        `json:"regularParticipantCanPublishMessage"`
	LastSeenDateTime                    null.Time   `json:"lastSeenDateTime"`
	RegularParticipantCanPinMessage     bool        `json:"regularParticipantCanPinMessage"`
	BlogAbout                           bool        `json:"blogAbout"`
	RegularParticipantCanWriteMessage   bool        `json:"regularParticipantCanWriteMessage"`
	CanWriteMessage                     bool        `json:"canWriteMessage"`
}

// designed to be able to get it from db with a single query
// it means without JOINs, without requesting aaa
type LightChatDto struct {
	Id                                  int64       `json:"id"`
	Name                                string      `json:"name"`
	Avatar                              null.String `json:"avatar"`
	AvatarBig                           null.String `json:"avatarBig"`
	LastUpdateDateTime                  time.Time   `json:"lastUpdateDateTime"`
	CanEdit                             null.Bool   `json:"canEdit"`
	CanDelete                           null.Bool   `json:"canDelete"`
	CanLeave                            null.Bool   `json:"canLeave"`
	CanBroadcast                        bool        `json:"canBroadcast"`
	CanVideoKick                        bool        `json:"canVideoKick"`
	CanChangeChatAdmins                 bool        `json:"canChangeChatAdmins"`
	IsTetATet                           bool        `json:"tetATet"`
	CanAudioMute                        bool        `json:"canAudioMute"`
	CanResend                           bool        `json:"canResend"`
	AvailableToSearch                   bool        `json:"availableToSearch"`
	IsResultFromSearch                  null.Bool   `json:"isResultFromSearch"`
	Blog                                bool        `json:"blog"`
	RegularParticipantCanPublishMessage bool        `json:"regularParticipantCanPublishMessage"`
	RegularParticipantCanPinMessage     bool        `json:"regularParticipantCanPinMessage"`
	BlogAbout                           bool        `json:"blogAbout"`
	RegularParticipantCanWriteMessage   bool        `json:"regularParticipantCanWriteMessage"`
	CanWriteMessage                     bool        `json:"canWriteMessage"`
}

// requires additional requests to aaa or to the different table
// TODO think about: we should be able to send (and apply) a portion of it, in order not to make unnecessary interactions during preparing a "full" Dto
type AdditionalChatDto struct {
	Id                int64       `json:"id"`
	ShortInfo         null.String `json:"shortInfo"`
	ParticipantIds    []int64     `json:"participantIds"`
	UnreadMessages    int64       `json:"unreadMessages"`
	ParticipantsCount int         `json:"participantsCount"`
	Pinned            bool        `json:"pinned"` // pinned for this particular user
	LoginColor        null.String `json:"loginColor"`
	LastSeenDateTime  null.Time   `json:"lastSeenDateTime"`
	Participants      []*User     `json:"participants"`
}

func (copied *BaseChatDto0) SetPersonalizedFields(admin bool, unreadMessages int64, participant bool) {
	copied.CanEdit = null.BoolFrom(admin && !copied.IsTetATet)
	copied.CanDelete = null.BoolFrom(admin)
	copied.CanLeave = null.BoolFrom(!admin && !copied.IsTetATet && participant)
	copied.UnreadMessages = unreadMessages
	copied.CanVideoKick = admin
	copied.CanAudioMute = admin
	copied.CanChangeChatAdmins = admin && !copied.IsTetATet
	copied.CanBroadcast = admin

	if !participant {
		copied.IsResultFromSearch = null.BoolFrom(true)
	}

	copied.CanWriteMessage = true
	// see also handlers PostMessage, EditMessage, DeleteMessage
	if !copied.RegularParticipantCanWriteMessage && !admin {
		copied.CanWriteMessage = false
	}
}

type ChatDeletedDto struct {
	Id int64 `json:"id"`
}

type ChatDto0 struct {
	BaseChatDto0
	Participants []*User `json:"participants"`
}

type ChatDtoWithTetATet interface {
	GetId() int64
	GetName() string
	GetAvatar() null.String
	GetIsTetATet() bool
	SetName(s string)
	SetAvatar(s null.String)
	SetShortInfo(s null.String)
	SetLoginColor(s null.String)
	SetLastSeenDateTime(t null.Time)
}

func (r *ChatDto0) GetId() int64 {
	return r.Id
}

func (r *ChatDto0) GetName() string {
	return r.Name
}

func (r *ChatDto0) SetName(s string) {
	r.Name = s
}

func (r *ChatDto0) GetIsTetATet() bool {
	return r.IsTetATet
}

func (r *BaseChatDto0) GetId() int64 {
	return r.Id
}

func (r *BaseChatDto0) GetName() string {
	return r.Name
}

func (r *BaseChatDto0) SetName(s string) {
	r.Name = s
}

func (r *BaseChatDto0) GetAvatar() null.String {
	return r.Avatar
}

func (r *BaseChatDto0) SetAvatar(s null.String) {
	r.Avatar = s
}

func (r *BaseChatDto0) SetShortInfo(s null.String) {
	r.ShortInfo = s
}

func (r *BaseChatDto0) SetLoginColor(s null.String) {
	r.LoginColor = s
}

func (r *BaseChatDto0) GetIsTetATet() bool {
	return r.IsTetATet
}

func (r *BaseChatDto0) SetLastSeenDateTime(t null.Time) {
	r.LastSeenDateTime = t
}

type ChatName struct {
	Name   string      `json:"name"`   // chatName or userName in case tet-a-tet
	Avatar null.String `json:"avatar"` // tet-a-tet -aware
	UserId int64       `json:"userId"` // userId chatName for
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
