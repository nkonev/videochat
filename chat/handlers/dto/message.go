package dto

import (
	"github.com/guregu/null"
	"time"
)

type DisplayMessageDto struct {
	Id             int64     `json:"id"`
	Text           string    `json:"text"`
	ChatId         int64     `json:"chatId"`
	OwnerId        int64     `json:"ownerId"`
	CreateDateTime time.Time `json:"createDateTime"`
	EditDateTime   null.Time `json:"editDateTime"`
	Owner          *User     `json:"owner"`
	CanEdit        bool      `json:"canEdit"`
}
