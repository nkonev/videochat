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

type VideoIsInvitingDto struct {
	ChatId       int64   `json:"chatId"`
	UserIds      []int64 `json:"userIds"` // invitee
	Status       bool    `json:"status"`  // true means inviting in process for this person(it sends it periodically), false means inviteng stopped (it is sent one time)
	BehalfUserId int64   `json:"behalfUserId"`
}
