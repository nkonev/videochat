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
	CanPin                              bool            `json:"canPin"`
	ConsiderMessagesAsUnread            bool            `json:"considerMessagesAsUnread"`
	AdditionalData                      *AdditionalData `json:"additionalData"`
}

type ChatDeletedDto struct {
	Id int64 `json:"id"`
}

type ChatDto struct {
	BaseChatDto
	Participants       []*User `json:"participants"`
	LastMessagePreview *string `json:"lastMessagePreview"`
}

type ChatUnreadMessageChanged struct {
	ChatId             int64     `json:"chatId"`
	UnreadMessages     int64     `json:"unreadMessages"`
	LastUpdateDateTime time.Time `json:"lastUpdateDateTime"`
}

type ChatNotificationSettingsChanged struct {
	ChatId                   int64 `json:"chatId"`
	ConsiderMessagesAsUnread bool  `json:"considerMessagesAsUnread"`
}
