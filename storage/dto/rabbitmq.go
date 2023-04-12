package dto

type ChatEvent struct {
	EventType         string             `json:"eventType"`
	ChatId            int64              `json:"chatId"`
	UserId            int64              `json:"userId"`
	FileUploadedEvent *FileUploadedEvent `json:"fileUploadedEvent"`
}

type FileUploadedEvent struct {
	Id            string  `json:"id"`
	Url           string  `json:"url"`
	PreviewUrl    *string `json:"previewUrl"`
	Type          *string `json:"aType"`
	CorrelationId *string `json:"correlationId"`
}
