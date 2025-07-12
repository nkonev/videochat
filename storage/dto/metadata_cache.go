package dto

import "time"

type MetadataCache struct {
	ChatId       int64
	FileItemUuid string
	Filename     string

	OwnerId             int64
	CorrelationId       *string
	ConferenceRecording *bool
	MessageRecording    *bool
	OriginalKey         *string

	Published bool

	CreateDateTime time.Time
}

type MetadataCacheId struct {
	ChatId       int64
	FileItemUuid string
	Filename     string
}
