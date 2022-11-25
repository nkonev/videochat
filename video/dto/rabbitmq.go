package dto

type GlobalEvent struct {
	EventType                 string                        `json:"eventType"`
	UserId                    int64                         `json:"userId"`
	VideoCallUserCountEvent   *VideoCallUserCountChangedDto `json:"videoCallUserCountEvent"`
	VideoChatInvitation       *VideoCallInvitation          `json:"videoCallInvitation"`
	VideoParticipantDialEvent *VideoDialChanges             `json:"videoParticipantDialEvent"`
	VideoCallRecordingEvent   *VideoCallRecordingChangedDto `json:"videoCallRecordingEvent"`
}

type NotificationEvent struct {
	EventType              string `json:"eventType"`
	ChatId                 int64  `json:"chatId"`
	UserId                 int64  `json:"userId"`
	MissedCallNotification bool   `json:"missedCallNotification"`
}
