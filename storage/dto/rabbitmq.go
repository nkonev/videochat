package dto

type ChatEvent struct {
	EventType           string               `json:"eventType"`
	ChatId              int64                `json:"chatId"`
	UserId              int64                `json:"userId"`
	PreviewCreatedEvent *PreviewCreatedEvent `json:"previewCreatedEvent"`
	FileEvent           *WrappedFileInfoDto  `json:"fileEvent"`
}

type PreviewCreatedEvent struct {
	Id            string  `json:"id"`
	Url           string  `json:"url"`
	PreviewUrl    *string `json:"previewUrl"`
	Type          *string `json:"aType"`
	CorrelationId *string `json:"correlationId"`
}
