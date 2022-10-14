package dto

type CentrifugeNotification struct {
	Payload   interface{} `json:"payload"`
	EventType string      `json:"type"`
}

type AllUnreadMessages struct {
	MessagesCount     int64      `json:"allUnreadMessages"`
}
