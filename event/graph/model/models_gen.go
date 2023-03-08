// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"time"

	"github.com/google/uuid"
)

type AllUnreadMessages struct {
	AllUnreadMessages int64 `json:"allUnreadMessages"`
}

type ChatDeletedDto struct {
	ID int64 `json:"id"`
}

type ChatDto struct {
	ID                  int64            `json:"id"`
	Name                string           `json:"name"`
	Avatar              *string          `json:"avatar"`
	AvatarBig           *string          `json:"avatarBig"`
	LastUpdateDateTime  time.Time        `json:"lastUpdateDateTime"`
	ParticipantIds      []int64          `json:"participantIds"`
	CanEdit             *bool            `json:"canEdit"`
	CanDelete           *bool            `json:"canDelete"`
	CanLeave            *bool            `json:"canLeave"`
	UnreadMessages      int64            `json:"unreadMessages"`
	CanBroadcast        bool             `json:"canBroadcast"`
	CanVideoKick        bool             `json:"canVideoKick"`
	CanChangeChatAdmins bool             `json:"canChangeChatAdmins"`
	TetATet             bool             `json:"tetATet"`
	CanAudioMute        bool             `json:"canAudioMute"`
	Participants        []*UserWithAdmin `json:"participants"`
	ParticipantsCount   int              `json:"participantsCount"`
	CanResend           bool             `json:"canResend"`
}

type ChatEvent struct {
	EventType             string                        `json:"eventType"`
	MessageEvent          *DisplayMessageDto            `json:"messageEvent"`
	MessageDeletedEvent   *MessageDeletedDto            `json:"messageDeletedEvent"`
	UserTypingEvent       *UserTypingDto                `json:"userTypingEvent"`
	MessageBroadcastEvent *MessageBroadcastNotification `json:"messageBroadcastEvent"`
	FileUploadedEvent     *FileUploadedEvent            `json:"fileUploadedEvent"`
	ParticipantsEvent     []*UserWithAdmin              `json:"participantsEvent"`
}

type ChatUnreadMessageChanged struct {
	ChatID         int64 `json:"chatId"`
	UnreadMessages int64 `json:"unreadMessages"`
}

type DisplayMessageDto struct {
	ID             int64                 `json:"id"`
	Text           string                `json:"text"`
	ChatID         int64                 `json:"chatId"`
	OwnerID        int64                 `json:"ownerId"`
	CreateDateTime time.Time             `json:"createDateTime"`
	EditDateTime   *time.Time            `json:"editDateTime"`
	Owner          *User                 `json:"owner"`
	CanEdit        bool                  `json:"canEdit"`
	CanDelete      bool                  `json:"canDelete"`
	FileItemUUID   *uuid.UUID            `json:"fileItemUuid"`
	EmbedMessage   *EmbedMessageResponse `json:"embedMessage"`
}

type EmbedMessageResponse struct {
	ID            int64   `json:"id"`
	ChatID        *int64  `json:"chatId"`
	ChatName      *string `json:"chatName"`
	Text          string  `json:"text"`
	Owner         *User   `json:"owner"`
	EmbedType     string  `json:"embedType"`
	IsParticipant bool    `json:"isParticipant"`
}

type FileUploadedEvent struct {
	ID            string  `json:"id"`
	URL           string  `json:"url"`
	PreviewURL    *string `json:"previewUrl"`
	AType         *string `json:"aType"`
	CorrelationID *string `json:"correlationId"`
}

type GlobalEvent struct {
	EventType                     string                    `json:"eventType"`
	ChatEvent                     *ChatDto                  `json:"chatEvent"`
	ChatDeletedEvent              *ChatDeletedDto           `json:"chatDeletedEvent"`
	UserEvent                     *User                     `json:"userEvent"`
	VideoUserCountChangedEvent    *VideoUserCountChangedDto `json:"videoUserCountChangedEvent"`
	VideoRecordingChangedEvent    *VideoRecordingChangedDto `json:"videoRecordingChangedEvent"`
	VideoCallInvitation           *VideoCallInvitationDto   `json:"videoCallInvitation"`
	VideoParticipantDialEvent     *VideoDialChanges         `json:"videoParticipantDialEvent"`
	UnreadMessagesNotification    *ChatUnreadMessageChanged `json:"unreadMessagesNotification"`
	AllUnreadMessagesNotification *AllUnreadMessages        `json:"allUnreadMessagesNotification"`
	NotificationEvent             *NotificationDto          `json:"notificationEvent"`
}

type MessageBroadcastNotification struct {
	Login  string `json:"login"`
	UserID int64  `json:"userId"`
	Text   string `json:"text"`
}

type MessageDeletedDto struct {
	ID     int64 `json:"id"`
	ChatID int64 `json:"chatId"`
}

type NotificationDto struct {
	ID               int64     `json:"id"`
	ChatID           int64     `json:"chatId"`
	MessageID        *int64    `json:"messageId"`
	NotificationType string    `json:"notificationType"`
	Description      string    `json:"description"`
	CreateDateTime   time.Time `json:"createDateTime"`
}

type User struct {
	ID     int64   `json:"id"`
	Login  string  `json:"login"`
	Avatar *string `json:"avatar"`
}

type UserOnline struct {
	ID     int64 `json:"id"`
	Online bool  `json:"online"`
}

type UserTypingDto struct {
	Login         string `json:"login"`
	ParticipantID int64  `json:"participantId"`
}

type UserWithAdmin struct {
	ID     int64   `json:"id"`
	Login  string  `json:"login"`
	Avatar *string `json:"avatar"`
	Admin  bool    `json:"admin"`
}

type VideoCallInvitationDto struct {
	ChatID   int64  `json:"chatId"`
	ChatName string `json:"chatName"`
}

type VideoDialChanged struct {
	UserID int64 `json:"userId"`
	Status bool  `json:"status"`
}

type VideoDialChanges struct {
	ChatID int64               `json:"chatId"`
	Dials  []*VideoDialChanged `json:"dials"`
}

type VideoRecordingChangedDto struct {
	RecordInProgress bool  `json:"recordInProgress"`
	ChatID           int64 `json:"chatId"`
}

type VideoUserCountChangedDto struct {
	UsersCount int64 `json:"usersCount"`
	ChatID     int64 `json:"chatId"`
}
