package dto

// used for notifications
type NotifyDto struct {
	UserId int64  `json:"userId"`
	Login  string `json:"login"`
}

type VideoInviteDto struct {
	ChatId       int64  `json:"chatId"`
	UserId       int64  `json:"userId"`
	BehalfUserId int64  `json:"behalfUserId"`
	BehalfLogin  string `json:"behalfLogin"`
}
