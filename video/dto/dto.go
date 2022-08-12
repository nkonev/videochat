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

// sent to chat through RabbitMQ
type ChatNotifyDto struct {
	Data       *NotifyDto `json:"data"`
	UsersCount int64      `json:"usersCount"`
	ChatId     int64      `json:"chatId"`
}

type VideoIsInvitingDto struct {
	ChatId       int64   `json:"chatId"`
	UserIds      []int64 `json:"userIds"` // invitee
	Status       bool    `json:"status"`  // true means inviting in process for this person(it sends it periodically), false means inviteng stopped (it is sent one time)
	BehalfUserId int64   `json:"behalfUserId"`
}
