package dto

type GlobalEvent struct {
	EventType                 string                        `json:"eventType"`
	UserId                    int64                         `json:"userId"`
	VideoCallUserCountEvent   *VideoCallUserCountChangedDto `json:"videoCallUserCountEvent"`
	VideoChatInvitation       *VideoCallInvitation          `json:"videoCallInvitation"`
	VideoParticipantDialEvent *VideoDialChanges             `json:"videoParticipantDialEvent"`
	VideoCallRecordingEvent   *VideoCallRecordingChangedDto `json:"videoCallRecordingEvent"`
}
