package dto

type UserSessionsKilledEvent struct {
	UserId int64 `json:"userId"`
	EventType string `json:"eventType"`
}

