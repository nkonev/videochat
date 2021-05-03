package dto

type CentrifugeNotification struct {
	Payload   interface{} `json:"payload"`
	EventType string      `json:"type"`
}

