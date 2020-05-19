package models

// db model
type Chat struct {
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	OwnerId int64  `json:"ownerId"`
}
