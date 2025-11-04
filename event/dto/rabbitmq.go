package dto

import (
	"github.com/montag451/go-eventbus"
	"time"
)

const CHAT_EVENTS = "events.chat"
const GLOBAL_USER_EVENTS = "events.user"
const USER_ONLINE = "user.online"

const GENERAL = "events.general"

const AAA_CREATE = "events.aaa.create"
const AAA_CHANGE = "events.aaa.change"
const AAA_DELETE = "events.aaa.delete"

const AAA_KILL_SESSIONS = "events.aaa.kill.sessions"

type PinnedMessageEvent struct {
	Message    PinnedMessageDto `json:"message"`
	TotalCount int64            `json:"totalCount"`
}

type WrappedFileInfoDto struct {
	FileInfoDto *FileInfoDto `json:"fileInfoDto"`
}

type PublishedMessageEvent struct {
	Message    PublishedMessageDto `json:"message"`
	TotalCount int64               `json:"totalCount"`
}

type ReactionChangedEvent struct {
	MessageId int64    `json:"messageId"`
	Reaction  Reaction `json:"reaction"`
}

type ChatEvent struct {
	TraceString                  string                        `json:"-"`
	EventType                    string                        `json:"eventType"`
	ChatId                       int64                         `json:"chatId"`
	UserId                       int64                         `json:"userId"`
	MessageNotification          *DisplayMessageDto            `json:"messageNotification"`
	MessageDeletedNotification   *MessageDeletedDto            `json:"messageDeletedNotification"`
	MessageBroadcastNotification *MessageBroadcastNotification `json:"messageBroadcastNotification"`
	PreviewCreatedEvent          *PreviewCreatedEvent          `json:"previewCreatedEvent"`
	Participants                 *[]*UserWithAdmin             `json:"participants"`
	PromoteMessageNotification   *PinnedMessageEvent           `json:"promoteMessageNotification"`
	FileEvent                    *WrappedFileInfoDto           `json:"fileEvent"`
	PublishedMessageNotification *PublishedMessageEvent        `json:"publishedMessageEvent"`
	ReactionChangedEvent         *ReactionChangedEvent         `json:"reactionChangedEvent"`
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
	ByAvatar         *string   `json:"byAvatar"`
	ChatTitle        string    `json:"chatTitle"`
}

type WrapperNotificationDto struct {
	NotificationDto NotificationDto `json:"notificationDto"`
	TotalCount      int64           `json:"totalCount"`
}

type HasUnreadMessagesChanged struct {
	HasUnreadMessages bool `json:"hasUnreadMessages"`
}

type BrowserNotification struct {
	ChatId      int64   `json:"chatId"`
	ChatName    string  `json:"chatName"`
	ChatAvatar  *string `json:"chatAvatar"`
	MessageId   int64   `json:"messageId"`
	MessageText string  `json:"messageText"`
	OwnerId     int64   `json:"ownerId"`
	OwnerLogin  string  `json:"ownerLogin"`
}

type GlobalUserEvent struct {
	TraceString                      string                          `json:"-"`
	EventType                        string                          `json:"eventType"`
	UserId                           int64                           `json:"userId"`
	ChatNotification                 *ChatDto                        `json:"chatNotification"`
	ChatDeletedDto                   *ChatDeletedDto                 `json:"chatDeletedNotification"`
	CoChattedParticipantNotification *User                           `json:"coChattedParticipantNotification"`
	VideoCallUserCountEvent          *VideoCallUserCountChangedDto   `json:"videoCallUserCountEvent"`
	VideoChatInvitation              *VideoCallInvitation            `json:"videoCallInvitation"`
	VideoParticipantDialEvent        *VideoDialChanges               `json:"videoParticipantDialEvent"`
	UnreadMessagesNotification       *ChatUnreadMessageChanged       `json:"unreadMessagesNotification"`
	AllUnreadMessagesNotification    *AllUnreadMessages              `json:"allUnreadMessagesNotification"`
	VideoCallRecordingEvent          *VideoCallRecordingChangedDto   `json:"videoCallRecordingEvent"`
	UserNotificationEvent            *WrapperNotificationDto         `json:"userNotificationEvent"`
	VideoCallScreenShareChangedDto   *VideoCallScreenShareChangedDto `json:"videoCallScreenShareChangedDto"`
	HasUnreadMessagesChanged         *HasUnreadMessagesChanged       `json:"hasUnreadMessagesChanged"`
	BrowserNotification              *BrowserNotification            `json:"browserNotification"`
	UserTypingNotification           *UserTypingNotification         `json:"userTypingNotification"`
}

func (GlobalUserEvent) Name() eventbus.EventName {
	return GLOBAL_USER_EVENTS
}

type PreviewCreatedEvent struct {
	Id            string  `json:"id"`
	Url           string  `json:"url"`
	PreviewUrl    *string `json:"previewUrl"`
	Type          *string `json:"aType"`
	CorrelationId string  `json:"correlationId"`
	FileItemUuid  string  `json:"fileItemUuid"`
}

type UserOnline struct {
	UserId           int64      `json:"userId"`
	Online           bool       `json:"online"`
	LastSeenDateTime *time.Time `json:"lastSeenDateTime"`
}

type ArrayUserOnline struct {
	UserOnlines []UserOnline
	TraceString string `json:"-"`
}

func (ArrayUserOnline) Name() eventbus.EventName {
	return USER_ONLINE
}

type FileInfoDto struct {
	Id             string    `json:"id"`
	Filename       string    `json:"filename"`
	Url            string    `json:"url"`
	PublishedUrl   *string   `json:"publishedUrl"`
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
	CanPlayAsAudio bool      `json:"canPlayAsAudio"`
	FileItemUuid   string    `json:"fileItemUuid"`
	CorrelationId  *string   `json:"correlationId"`
	Previewable    bool      `json:"previewable"`
	Type           *string   `json:"aType"`
}

type GeneralEvent struct {
	TraceString                          string                              `json:"-"`
	EventType                            string                              `json:"eventType"`
	VideoCallUsersCallStatusChangedEvent *VideoCallUsersCallStatusChangedDto `json:"videoCallUsersCallStatusChangedEvent"`
}

func (GeneralEvent) Name() eventbus.EventName {
	return GENERAL
}
