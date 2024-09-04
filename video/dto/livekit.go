package dto

import "github.com/google/uuid"

type MetadataDto struct {
	UserId int64  `json:"userId"`
	Login  string `json:"login"`
	Avatar string `json:"avatar"` // url
	TokenId uuid.UUID `json:"tokenId"`
}
