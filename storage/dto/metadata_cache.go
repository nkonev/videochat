package dto

import "time"

type MetadataCache struct {
	ChatId       int64
	FileItemUuid string
	Filename     string

	OwnerId       int64
	CorrelationId *string

	Published bool

	CreateDateTime time.Time
}

type MetadataCacheId struct {
	ChatId       int64
	FileItemUuid string
	Filename     string
}
