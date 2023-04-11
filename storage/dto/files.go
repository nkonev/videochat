package dto

import (
	"github.com/google/uuid"
	"time"
)

type FileInfoDto struct {
	Id           uuid.UUID `json:"id"`
	Filename     string    `json:"filename"`
	Url          string    `json:"url"`
	PreviewUrl   *string   `json:"previewUrl"`
	Size         int64     `json:"size"`
	CanDelete    bool      `json:"canDelete"`
	CanEdit      bool      `json:"canEdit"`
	CanShare     bool      `json:"canShare"`
	LastModified time.Time `json:"lastModified"`
	OwnerId      int64     `json:"ownerId"`
	Owner        *User     `json:"owner"`
}

type MinioEvent struct {
	EventName     string
	Key           string
	ChatId        int64
	OwnerId       int64
	CorrelationId *string
	FileId        uuid.UUID
}
