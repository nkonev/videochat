package dto

import "time"

type MetadataCache struct {
	ChatId       int64
	FileItemUuid string
	Filename     string

	OwnerId       int64
	CorrelationId *string

	Published bool

	FileSize int64

	CreateDateTime time.Time
	EditDateTime   time.Time
}

type MetadataCacheId struct {
	ChatId       int64
	FileItemUuid string
	Filename     string
}
