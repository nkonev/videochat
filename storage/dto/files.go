package dto

import (
	"time"
)

type FileInfoDto struct {
	Id             string    `json:"id"`
	Filename       string    `json:"filename"`
	Url            string    `json:"url"`
	PublicUrl      *string   `json:"publicUrl"`
	PreviewUrl     *string   `json:"previewUrl"`
	Size           int64     `json:"size"`
	CanDelete      bool      `json:"canDelete"`
	CanEdit        bool      `json:"canEdit"`
	CanShare       bool      `json:"canShare"`
	LastModified   time.Time `json:"lastModified"`
	OwnerId        int64     `json:"ownerId"`
	Owner          *User     `json:"owner"`
	CanPlayAsVideo bool      `json:"canPlayAsVideo"`
	CanShowAsImage bool      `json:"canShowAsImage"`
	CanPlayAsAudio bool      `json:"canPlayAsAudio"`
	FileItemUuid   string `json:"fileItemUuid"`
}

type WrappedFileInfoDto struct {
	FileInfoDto *FileInfoDto `json:"fileInfoDto"`
}

type MinioEvent struct {
	EventName     string
	Key           string
	ChatId        int64
	OwnerId       int64
	CorrelationId *string
}
