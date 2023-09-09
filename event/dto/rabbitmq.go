package dto

import (
	"github.com/google/uuid"
	"github.com/montag451/go-eventbus"
	"time"
)

const CHAT_EVENTS = "events.chat"
const GLOBAL_EVENTS = "events.global"
const USER_ONLINE = "user.online"

type PinnedMessageEvent struct {
	Message    DisplayMessageDto `json:"message"`
	TotalCount int64             `json:"totalCount"`
}

type WrappedFileInfoDto struct {
	FileInfoDto *FileInfoDto `json:"fileInfoDto"`
	Count       int64        `json:"count"`
	FileItemUuid uuid.UUID   `json:"fileItemUuid"`
}

type ChatEvent struct {
	EventType                    string                        `json:"eventType"`
	ChatId                       int64                         `json:"chatId"`
	UserId                       int64                         `json:"userId"`
	MessageNotification          *DisplayMessageDto            `json:"messageNotification"`
	MessageDeletedNotification   *MessageDeletedDto            `json:"messageDeletedNotification"`
	UserTypingNotification       *UserTypingNotification       `json:"userTypingNotification"`
	MessageBroadcastNotification *MessageBroadcastNotification `json:"messageBroadcastNotification"`
	PreviewCreatedEvent          *PreviewCreatedEvent          `json:"previewCreatedEvent"`
	Participants                 *[]*UserWithAdmin             `json:"participants"`
	PromoteMessageNotification   *PinnedMessageEvent           `json:"promoteMessageNotification"`
	FileEvent                    *WrappedFileInfoDto           `json:"fileEvent"`
}

func (ChatEvent) Name() eventbus.EventName {
	return CHAT_EVENTS
}

type NotificationDto struct {
	Id               int64     `json:"id"`
	ChatId           int64     `json:"chatId"`
	MessageId        *int64    `json:"messageId"`
	NotificationType string    `json:"notificationType"`
	Description      string    `json:"description"`
	CreateDateTime   time.Time `json:"createDateTime"`
	ByUserId         int64     `json:"byUserId"`
	ByLogin          string    `json:"byLogin"`
	ChatTitle        string    `json:"chatTitle"`
}

type GlobalEvent struct {
	EventType                     string                        `json:"eventType"`
	UserId                        int64                         `json:"userId"`
	ChatNotification              *ChatDtoWithAdmin             `json:"chatNotification"`
	ChatDeletedDto                *ChatDeletedDto               `json:"chatDeletedNotification"`
	UserProfileNotification       *User                         `json:"userProfileNotification"`
	VideoCallUserCountEvent       *VideoCallUserCountChangedDto `json:"videoCallUserCountEvent"`
	VideoChatInvitation           *VideoCallInvitation          `json:"videoCallInvitation"`
	VideoParticipantDialEvent     *VideoDialChanges             `json:"videoParticipantDialEvent"`
	UnreadMessagesNotification    *ChatUnreadMessageChanged     `json:"unreadMessagesNotification"`
	AllUnreadMessagesNotification *AllUnreadMessages            `json:"allUnreadMessagesNotification"`
	VideoCallRecordingEvent       *VideoCallRecordingChangedDto `json:"videoCallRecordingEvent"`
	UserNotificationEvent         *NotificationDto              `json:"userNotificationEvent"`
	VideoCallScreenShareChangedDto *VideoCallScreenShareChangedDto `json:"videoCallScreenShareChangedDto"`
}

func (GlobalEvent) Name() eventbus.EventName {
	return GLOBAL_EVENTS
}

type PreviewCreatedEvent struct {
	Id            string  `json:"id"`
	Url           string  `json:"url"`
	PreviewUrl    *string `json:"previewUrl"`
	Type          *string `json:"aType"`
	CorrelationId string  `json:"correlationId"`
}

type UserOnline struct {
	UserId int64 `json:"userId"`
	Online bool  `json:"online"`
}

type ArrayUserOnline []UserOnline

func (ArrayUserOnline) Name() eventbus.EventName {
	return USER_ONLINE
}

type FileInfoDto struct {
	Id             string    `json:"id"`
	Filename       string    `json:"filename"`
	Url            string    `json:"url"`
	PublicUrl      *string   `json:"publicUrl"`
	PreviewUrl     *string   `json:"previewUrl"`
	Size           int64     `json:"size"`
	CanDelete      bool      `json:"canDelete"`
	CanEdit        bool      `json:"canEdit"`
	CanShare       bool      `json:"canShare"`
	LastModified   time.Time `json:"lastModified"`
	OwnerId        int64     `json:"ownerId"`
	Owner          *User     `json:"owner"`
	CanPlayAsVideo bool      `json:"canPlayAsVideo"`
	CanShowAsImage bool      `json:"canShowAsImage"`
}
