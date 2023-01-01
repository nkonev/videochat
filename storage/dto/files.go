package dto

import (
	"time"
)

type FileInfoDto struct {
	Id           string    `json:"id"`
	Filename     string    `json:"filename"`
	Url          string    `json:"url"`
	PublicUrl    *string   `json:"publicUrl"`
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
	CorrelationId string
}
