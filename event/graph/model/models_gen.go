// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"time"
)

type UserAccountEventDto interface {
	IsUserAccountEventDto()
}

type AllUnreadMessages struct {
	AllUnreadMessages int64 `json:"allUnreadMessages"`
}

type BrowserNotification struct {
	ChatID      int64   `json:"chatId"`
	ChatName    string  `json:"chatName"`
	ChatAvatar  *string `json:"chatAvatar"`
	MessageID   int64   `json:"messageId"`
	MessageText string  `json:"messageText"`
	OwnerID     int64   `json:"ownerId"`
	OwnerLogin  string  `json:"ownerLogin"`
}

type ChatDeletedDto struct {
	ID int64 `json:"id"`
}

type ChatDto struct {
	ID                                  int64                   `json:"id"`
	Name                                string                  `json:"name"`
	Avatar                              *string                 `json:"avatar"`
	AvatarBig                           *string                 `json:"avatarBig"`
	ShortInfo                           *string                 `json:"shortInfo"`
	LastUpdateDateTime                  time.Time               `json:"lastUpdateDateTime"`
	ParticipantIds                      []int64                 `json:"participantIds"`
	CanEdit                             *bool                   `json:"canEdit"`
	CanDelete                           *bool                   `json:"canDelete"`
	CanLeave                            *bool                   `json:"canLeave"`
	UnreadMessages                      int64                   `json:"unreadMessages"`
	CanBroadcast                        bool                    `json:"canBroadcast"`
	CanVideoKick                        bool                    `json:"canVideoKick"`
	CanChangeChatAdmins                 bool                    `json:"canChangeChatAdmins"`
	TetATet                             bool                    `json:"tetATet"`
	CanAudioMute                        bool                    `json:"canAudioMute"`
	Participants                        []*ParticipantWithAdmin `json:"participants"`
	ParticipantsCount                   int                     `json:"participantsCount"`
	CanResend                           bool                    `json:"canResend"`
	AvailableToSearch                   bool                    `json:"availableToSearch"`
	IsResultFromSearch                  *bool                   `json:"isResultFromSearch"`
	Pinned                              bool                    `json:"pinned"`
	Blog                                bool                    `json:"blog"`
	LoginColor                          *string                 `json:"loginColor"`
	RegularParticipantCanPublishMessage bool                    `json:"regularParticipantCanPublishMessage"`
}

type ChatEvent struct {
	EventType             string                        `json:"eventType"`
	MessageEvent          *DisplayMessageDto            `json:"messageEvent"`
	MessageDeletedEvent   *MessageDeletedDto            `json:"messageDeletedEvent"`
	UserTypingEvent       *UserTypingDto                `json:"userTypingEvent"`
	MessageBroadcastEvent *MessageBroadcastNotification `json:"messageBroadcastEvent"`
	PreviewCreatedEvent   *PreviewCreatedEvent          `json:"previewCreatedEvent"`
	ParticipantsEvent     []*ParticipantWithAdmin       `json:"participantsEvent"`
	PromoteMessageEvent   *PinnedMessageEvent           `json:"promoteMessageEvent"`
	FileEvent             *WrappedFileInfoDto           `json:"fileEvent"`
	PublishedMessageEvent *PublishedMessageEvent        `json:"publishedMessageEvent"`
	ReactionChangedEvent  *ReactionChangedEvent         `json:"reactionChangedEvent"`
}

type ChatUnreadMessageChanged struct {
	ChatID             int64     `json:"chatId"`
	UnreadMessages     int64     `json:"unreadMessages"`
	LastUpdateDateTime time.Time `json:"lastUpdateDateTime"`
}

type DataDto struct {
	Enabled   bool     `json:"enabled"`
	Expired   bool     `json:"expired"`
	Locked    bool     `json:"locked"`
	Confirmed bool     `json:"confirmed"`
	Roles     []string `json:"roles"`
}

type DisplayMessageDto struct {
	ID             int64                 `json:"id"`
	Text           string                `json:"text"`
	ChatID         int64                 `json:"chatId"`
	OwnerID        int64                 `json:"ownerId"`
	CreateDateTime time.Time             `json:"createDateTime"`
	EditDateTime   *time.Time            `json:"editDateTime"`
	Owner          *Participant          `json:"owner"`
	CanEdit        bool                  `json:"canEdit"`
	CanDelete      bool                  `json:"canDelete"`
	FileItemUUID   *string               `json:"fileItemUuid"`
	EmbedMessage   *EmbedMessageResponse `json:"embedMessage"`
	Pinned         bool                  `json:"pinned"`
	BlogPost       bool                  `json:"blogPost"`
	PinnedPromoted *bool                 `json:"pinnedPromoted"`
	Reactions      []*Reaction           `json:"reactions"`
	Published      bool                  `json:"published"`
	CanPublish     bool                  `json:"canPublish"`
}

type EmbedMessageResponse struct {
	ID            int64        `json:"id"`
	ChatID        *int64       `json:"chatId"`
	ChatName      *string      `json:"chatName"`
	Text          string       `json:"text"`
	Owner         *Participant `json:"owner"`
	EmbedType     string       `json:"embedType"`
	IsParticipant bool         `json:"isParticipant"`
}

type FileInfoDto struct {
	ID             string       `json:"id"`
	Filename       string       `json:"filename"`
	URL            string       `json:"url"`
	PublicURL      *string      `json:"publicUrl"`
	PreviewURL     *string      `json:"previewUrl"`
	Size           int64        `json:"size"`
	CanDelete      bool         `json:"canDelete"`
	CanEdit        bool         `json:"canEdit"`
	CanShare       bool         `json:"canShare"`
	LastModified   time.Time    `json:"lastModified"`
	OwnerID        int64        `json:"ownerId"`
	Owner          *Participant `json:"owner"`
	CanPlayAsVideo bool         `json:"canPlayAsVideo"`
	CanShowAsImage bool         `json:"canShowAsImage"`
	CanPlayAsAudio bool         `json:"canPlayAsAudio"`
	FileItemUUID   string       `json:"fileItemUuid"`
}

type ForceLogoutEvent struct {
	ReasonType string `json:"reasonType"`
}

type GlobalEvent struct {
	EventType                      string                          `json:"eventType"`
	ChatEvent                      *ChatDto                        `json:"chatEvent"`
	ChatDeletedEvent               *ChatDeletedDto                 `json:"chatDeletedEvent"`
	CoChattedParticipantEvent      *Participant                    `json:"coChattedParticipantEvent"`
	VideoUserCountChangedEvent     *VideoUserCountChangedDto       `json:"videoUserCountChangedEvent"`
	VideoRecordingChangedEvent     *VideoRecordingChangedDto       `json:"videoRecordingChangedEvent"`
	VideoCallInvitation            *VideoCallInvitationDto         `json:"videoCallInvitation"`
	VideoParticipantDialEvent      *VideoDialChanges               `json:"videoParticipantDialEvent"`
	UnreadMessagesNotification     *ChatUnreadMessageChanged       `json:"unreadMessagesNotification"`
	AllUnreadMessagesNotification  *AllUnreadMessages              `json:"allUnreadMessagesNotification"`
	NotificationEvent              *WrapperNotificationDto         `json:"notificationEvent"`
	VideoCallScreenShareChangedDto *VideoCallScreenShareChangedDto `json:"videoCallScreenShareChangedDto"`
	ForceLogout                    *ForceLogoutEvent               `json:"forceLogout"`
	HasUnreadMessagesChanged       *HasUnreadMessagesChangedEvent  `json:"hasUnreadMessagesChanged"`
	BrowserNotification            *BrowserNotification            `json:"browserNotification"`
}

type HasUnreadMessagesChangedEvent struct {
	HasUnreadMessages bool `json:"hasUnreadMessages"`
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
	ByUserID         int64     `json:"byUserId"`
	ByLogin          string    `json:"byLogin"`
	ByAvatar         *string   `json:"byAvatar"`
	ChatTitle        string    `json:"chatTitle"`
}

type OAuth2Identifiers struct {
	FacebookID  *string `json:"facebookId"`
	VkontakteID *string `json:"vkontakteId"`
	GoogleID    *string `json:"googleId"`
	KeycloakID  *string `json:"keycloakId"`
}

type Participant struct {
	ID         int64   `json:"id"`
	Login      string  `json:"login"`
	Avatar     *string `json:"avatar"`
	ShortInfo  *string `json:"shortInfo"`
	LoginColor *string `json:"loginColor"`
}

type ParticipantWithAdmin struct {
	ID         int64   `json:"id"`
	Login      string  `json:"login"`
	Avatar     *string `json:"avatar"`
	Admin      bool    `json:"admin"`
	ShortInfo  *string `json:"shortInfo"`
	LoginColor *string `json:"loginColor"`
}

type PinnedMessageDto struct {
	ID             int64        `json:"id"`
	Text           string       `json:"text"`
	ChatID         int64        `json:"chatId"`
	OwnerID        int64        `json:"ownerId"`
	Owner          *Participant `json:"owner"`
	PinnedPromoted bool         `json:"pinnedPromoted"`
	CreateDateTime time.Time    `json:"createDateTime"`
}

type PinnedMessageEvent struct {
	Message *PinnedMessageDto `json:"message"`
	Count   int64             `json:"count"`
}

type PreviewCreatedEvent struct {
	ID            string  `json:"id"`
	URL           string  `json:"url"`
	PreviewURL    *string `json:"previewUrl"`
	AType         *string `json:"aType"`
	CorrelationID *string `json:"correlationId"`
}

type PublishedMessageDto struct {
	ID             int64        `json:"id"`
	Text           string       `json:"text"`
	ChatID         int64        `json:"chatId"`
	OwnerID        int64        `json:"ownerId"`
	Owner          *Participant `json:"owner"`
	CanPublish     bool         `json:"canPublish"`
	CreateDateTime time.Time    `json:"createDateTime"`
}

type PublishedMessageEvent struct {
	Message *PublishedMessageDto `json:"message"`
	Count   int64                `json:"count"`
}

type Query struct {
}

type Reaction struct {
	Count    int64          `json:"count"`
	Users    []*Participant `json:"users"`
	Reaction string         `json:"reaction"`
}

type ReactionChangedEvent struct {
	MessageID int64     `json:"messageId"`
	Reaction  *Reaction `json:"reaction"`
}

type Subscription struct {
}

type UserAccountDto struct {
	ID                int64              `json:"id"`
	Login             string             `json:"login"`
	Avatar            *string            `json:"avatar"`
	AvatarBig         *string            `json:"avatarBig"`
	ShortInfo         *string            `json:"shortInfo"`
	LastLoginDateTime *time.Time         `json:"lastLoginDateTime"`
	Oauth2Identifiers *OAuth2Identifiers `json:"oauth2Identifiers"`
	LoginColor        *string            `json:"loginColor"`
}

func (UserAccountDto) IsUserAccountEventDto() {}

type UserAccountEvent struct {
	EventType        string              `json:"eventType"`
	UserAccountEvent UserAccountEventDto `json:"userAccountEvent"`
}

type UserAccountExtendedDto struct {
	ID                int64              `json:"id"`
	Login             string             `json:"login"`
	Avatar            *string            `json:"avatar"`
	AvatarBig         *string            `json:"avatarBig"`
	ShortInfo         *string            `json:"shortInfo"`
	LastLoginDateTime *time.Time         `json:"lastLoginDateTime"`
	Oauth2Identifiers *OAuth2Identifiers `json:"oauth2Identifiers"`
	AdditionalData    *DataDto           `json:"additionalData"`
	CanLock           bool               `json:"canLock"`
	CanDelete         bool               `json:"canDelete"`
	CanChangeRole     bool               `json:"canChangeRole"`
	CanConfirm        bool               `json:"canConfirm"`
	LoginColor        *string            `json:"loginColor"`
	CanRemoveSessions bool               `json:"canRemoveSessions"`
}

func (UserAccountExtendedDto) IsUserAccountEventDto() {}

type UserDeletedDto struct {
	ID int64 `json:"id"`
}

func (UserDeletedDto) IsUserAccountEventDto() {}

type UserStatusEvent struct {
	UserID    int64  `json:"userId"`
	Online    *bool  `json:"online"`
	IsInVideo *bool  `json:"isInVideo"`
	EventType string `json:"eventType"`
}

type UserTypingDto struct {
	Login         string `json:"login"`
	ParticipantID int64  `json:"participantId"`
}

type VideoCallInvitationDto struct {
	ChatID   int64   `json:"chatId"`
	ChatName string  `json:"chatName"`
	Status   string  `json:"status"`
	Avatar   *string `json:"avatar"`
}

type VideoCallScreenShareChangedDto struct {
	ChatID          int64 `json:"chatId"`
	HasScreenShares bool  `json:"hasScreenShares"`
}

type VideoDialChanged struct {
	UserID int64  `json:"userId"`
	Status string `json:"status"`
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

type WrappedFileInfoDto struct {
	FileInfoDto *FileInfoDto `json:"fileInfoDto"`
}

type WrapperNotificationDto struct {
	Count           int64            `json:"count"`
	NotificationDto *NotificationDto `json:"notificationDto"`
}
