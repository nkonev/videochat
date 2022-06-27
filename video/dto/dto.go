package dto

// used for notifications
type NotifyDto struct {
	UserId int64  `json:"userId"`
	Login  string `json:"login"`
}
