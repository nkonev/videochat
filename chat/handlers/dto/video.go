package dto

type NotifyRequest struct {
	UserId int64  `json:"userId"`
	Login  string `json:"login"`
}
type ChatNotifyDto struct {
	Data       *NotifyRequest `json:"data"`
	UsersCount int64          `json:"usersCount"`
	ChatId     int64          `json:"chatId"`
}
type VideoInviteDto struct {
	ChatId       int64  `json:"chatId"`
	UserId       int64  `json:"userId"`
	BehalfUserId int64  `json:"behalfUserId"`
	BehalfLogin  string `json:"behalfLogin"`
}
