package dto

import (
	"time"
)

const NoChatTitle = ""
const NoSearchString = ""
const ReservedPublicallyAvailableForSearchChats = "__AVAILABLE_FOR_SEARCH"
const DefaultCanReact = true
const DefaultCanResend = false
const DefaultAvailableToSearch = false
const DefaultRegularParticipantCanWriteMessage = true
const DefaultRegularParticipantCanPublishMessage = false
const DefaultRegularParticipantCanPinMessage = false
const DefaultRegularParticipantCanAddParticipant = true

type ChatViewDto struct {
	Id                                  int64      `json:"id"`
	BehalfUserId                        int64      `json:"-"` // behalf user id
	Title                               string     `json:"name"`
	Pinned                              bool       `json:"pinned"`
	UnreadMessages                      int64      `json:"unreadMessages"`
	LastMessageId                       *int64     `json:"lastMessageId"`
	LastMessageOwnerId                  *int64     `json:"lastMessageOwnerId"`
	LastMessageContent                  *string    `json:"lastMessageContent"`
	ParticipantsCount                   int64      `json:"participantsCount"`
	ParticipantIds                      []int64    `json:"participantIds"` // ids of last N participants
	Blog                                bool       `json:"blog"`
	BlogAbout                           bool       `json:"blogAbout"`
	UpdateDateTime                      *time.Time `json:"lastUpdateDateTime"` // for sake compatibility
	TetATet                             bool       `json:"tetATet"`
	Avatar                              *string    `json:"avatar"`
	AvatarBig                           *string    `json:"avatarBig"`
	ConsiderMessagesAsUnread            bool       `json:"considerMessagesAsUnread"`
	CanResend                           bool       `json:"canResend"`
	CanReact                            bool       `json:"canReact"`
	CanPin                              bool       `json:"canPin"`
	RegularParticipantCanPublishMessage bool       `json:"regularParticipantCanPublishMessage"`
	RegularParticipantCanPinMessage     bool       `json:"regularParticipantCanPinMessage"`
	RegularParticipantCanWriteMessage   bool       `json:"regularParticipantCanWriteMessage"`
	AvailableToSearch                   bool       `json:"availableToSearch"`
	IsParticipant                       bool       `json:"-"`
	RegularParticipantCanAddParticipant bool       `json:"regularParticipantCanAddParticipant"`
}

type ChatId struct {
	Pinned             bool
	LastUpdateDateTime time.Time
	Id                 int64
}

type ChatBaseCreateDto struct {
	Title                               string  `json:"name"`
	ParticipantIds                      []int64 `json:"participantIds"`
	Blog                                bool    `json:"blog"`
	BlogAbout                           bool    `json:"blogAbout"`
	Avatar                              *string `json:"avatar"`
	AvatarBig                           *string `json:"avatarBig"`
	CanReact                            *bool   `json:"canReact"`
	RegularParticipantCanWriteMessage   *bool   `json:"regularParticipantCanWriteMessage"`
	CanResend                           *bool   `json:"canResend"`
	AvailableToSearch                   *bool   `json:"availableToSearch"`
	RegularParticipantCanPublishMessage *bool   `json:"regularParticipantCanPublishMessage"`
	RegularParticipantCanPinMessage     *bool   `json:"regularParticipantCanPinMessage"`
	RegularParticipantCanAddParticipant *bool   `json:"regularParticipantCanAddParticipant"`
}

type ChatCreateDto struct {
	ChatBaseCreateDto
}

type ChatEditDto struct {
	Id int64 `json:"id"`
	ChatBaseCreateDto
}

type ChatDeletedDto struct {
	Id int64 `json:"id"`
}

type ChatTetATetUpsertedDto struct {
	ChatId int64 `json:"chatId"`
}

type ChatViewEnrichedDto struct {
	ChatViewDto
	LastMessagePreview  *string `json:"lastMessagePreview"`
	Participants        []User  `json:"participants"`
	CanEdit             *bool   `json:"canEdit"`
	CanDelete           *bool   `json:"canDelete"`
	CanLeave            *bool   `json:"canLeave"`
	CanBroadcast        bool    `json:"canBroadcast"`
	CanVideoKick        bool    `json:"canVideoKick"`
	CanAudioMute        bool    `json:"canAudioMute"`
	CanChangeChatAdmins bool    `json:"canChangeChatAdmins"`
	IsResultFromSearch  bool    `json:"isResultFromSearch"` // is result os "search publically available to join"
	CanWriteMessage     bool    `json:"canWriteMessage"`
	CanAddParticipant   bool    `json:"canAddParticipant"`

	LastSeenDateTime *time.Time      `json:"lastSeenDateTime"`
	ShortInfo        *string         `json:"shortInfo"`
	LoginColor       *string         `json:"loginColor"`
	AdditionalData   *AdditionalData `json:"additionalData"`
}

type GetChatsResponseDto struct {
	Items   []ChatViewEnrichedDto `json:"items"`
	HasNext bool                  `json:"hasNext"`
}

type ChatBasic struct {
	Id                                  int64   `db:"id"`
	Title                               string  `db:"title"`
	Avatar                              *string `db:"avatar"`
	CanResend                           bool    `db:"can_resend"`
	TetATet                             bool    `db:"tet_a_tet"`
	IsBlog                              bool    `db:"blog"`
	AvailableToSearch                   bool    `db:"available_to_search"`
	RegularParticipantCanPublishMessage bool    `db:"regular_participant_can_publish_message"`
	RegularParticipantCanPinMessage     bool    `db:"regular_participant_can_pin_message"`
	RegularParticipantCanWriteMessage   bool    `db:"regular_participant_can_write_message"`
}

type BasicChatDtoExtended struct {
	ChatBasic
	BehalfUserId            int64 `db:"user_id"`
	BehalfUserIsParticipant bool  `db:"behalf_user_is_participant"`
}

type UserChatNotificationSettings struct {
	ConsiderMessagesOfThisChatAsUnread bool `json:"considerMessagesOfThisChatAsUnread" db:"consider_messages_as_unread"`
}

type ChatUserViewBasic struct {
	ChatId         int64     `db:"id"`
	UpdateDateTime time.Time `db:"update_date_time"`
	UnreadMessages int64     `db:"unread_messages"`
}

type ChatExists struct {
	Exists bool  `json:"exists"`
	ChatId int64 `json:"chatId"`
}
type ChatFilterDto struct {
	SearchString string `json:"searchString"`
	ChatId       int64  `json:"chatId"` // id of probe element
}

type ChatAuthorizationData struct {
	IsChatFound                          bool `db:"is_chat_found"`
	IsParticipant                        bool `db:"is_chat_participant"`
	IsChatAdmin                          bool `db:"is_chat_admin"`
	ChatCanWriteMessage                  bool `db:"chat_can_write_message"`
	ChatCanResendMessage                 bool `db:"chat_can_resend_message"`
	ChatCanReactOnMessage                bool `db:"chat_can_react_on_message"`
	ChatIsTetATet                        bool `db:"chat_is_tet_a_tet"`
	AvailableToSearch                    bool `db:"chat_is_available_to_search"`
	IsBlog                               bool `db:"chat_is_blog"`
	RegularParticipantCanAddParticipants bool `db:"regular_participant_can_add_participant"`
}

type ChatNotificationSettingsChanged struct {
	ChatId                   int64 `json:"chatId"`
	ConsiderMessagesAsUnread bool  `json:"considerMessagesAsUnread"`
}

type ChatInfoForNotification struct {
	ChatName   string  `db:"title"`
	ChatAvatar *string `db:"avatar"`
}

type ChatParticipant struct {
	ChatId int64 `db:"chat_id"`
	UserId int64 `db:"user_id"`
}

type BasicChatDto struct {
	TetATet        bool    `json:"tetATet"`
	ParticipantIds []int64 `json:"participantIds"`
}

type ChatName struct {
	Name   string  `json:"name"`   // chatName or userName in case tet-a-tet
	Avatar *string `json:"avatar"` // tet-a-tet -aware
	UserId int64   `json:"userId"` // userId chatName for
}

type ParticipantBelongsToChat struct {
	UserId  int64 `json:"userId"`
	Belongs bool  `json:"belongs"`
}

type ParticipantsBelongToChat struct {
	Users []ParticipantBelongsToChat `json:"users"`
}
