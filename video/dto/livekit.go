package dto

type MetadataDto struct {
	UserId int64  `json:"userId"`
	Login  string `json:"login"`
	Avatar string `json:"avatar"` // url
}
