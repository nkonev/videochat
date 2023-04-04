package dto

type GlobalEvent struct {
	EventType                 string                        `json:"eventType"`
	UserId                    int64                         `json:"userId"`
	VideoCallUserCountEvent   *VideoCallUserCountChangedDto `json:"videoCallUserCountEvent"`
	VideoChatInvitation       *VideoCallInvitation          `json:"videoCallInvitation"`
	VideoParticipantDialEvent *VideoDialChanges             `json:"videoParticipantDialEvent"`
	VideoCallRecordingEvent   *VideoCallRecordingChangedDto `json:"videoCallRecordingEvent"`
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
