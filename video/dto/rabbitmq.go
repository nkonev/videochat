package dto

type UserEvent struct {
	EventType                 string                        `json:"eventType"`
	UserId                    int64                         `json:"userId"`
	VideoCallUserCountEvent   *VideoCallUserCountChangedDto `json:"videoCallUserCountEvent"`
	VideoChatInvitation       *VideoCallInvitation          `json:"videoCallInvitation"`
	VideoParticipantDialEvent *VideoDialChanges             `json:"videoParticipantDialEvent"`
	VideoCallRecordingEvent   *VideoCallRecordingChangedDto `json:"videoCallRecordingEvent"`
	VideoCallScreenShareChangedDto *VideoCallScreenShareChangedDto `json:"videoCallScreenShareChangedDto"`
}

type MissedCallNotification struct {
	Description string `json:"description"`
}

type NotificationEvent struct {
	EventType              string                  `json:"eventType"`
	ChatId                 int64                   `json:"chatId"`
	UserId                 int64                   `json:"userId"`
	ByUserId               int64                   `json:"byUserId"`
	ByLogin                string                  `json:"byLogin"`
	MissedCallNotification *MissedCallNotification `json:"missedCallNotification"`
}

type GeneralEvent struct {
	EventType string `json:"eventType"`
	VideoCallUsersCallStatusChangedEvent *VideoCallUsersCallStatusChangedDto `json:"videoCallUsersCallStatusChangedEvent"`
}
