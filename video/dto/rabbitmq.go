package dto

type GlobalEvent struct {
	EventType         string               `json:"eventType"`
	UserId            int64                `json:"userId"`
	VideoNotification *VideoCallChangedDto `json:"videoNotification"`
}
