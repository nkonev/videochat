package dto

import (
	"fmt"
	"time"
)

const NoFileItemUuid = ""
const NoChatId = -1

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

func (mcid *MetadataCacheId) String() string {
	return fmt.Sprintf("MetadataCacheId{chatId=%v, fileItemUuid=%v, filename=%v}", mcid.ChatId, mcid.FileItemUuid, mcid.Filename)
}
