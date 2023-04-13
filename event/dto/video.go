package dto

type VideoCallUserCountChangedDto struct {
	UsersCount int64 `json:"usersCount"`
	ChatId     int64 `json:"chatId"`
}

type VideoInviteDto struct {
	ChatId       int64   `json:"chatId"`
	UserIds      []int64 `json:"userIds"`
	BehalfUserId int64   `json:"behalfUserId"`
	BehalfLogin  string  `json:"behalfLogin"`
}

type VideoCallInvitation struct {
	ChatId   int64  `json:"chatId"`
	ChatName string `json:"chatName"`
}

type VideoDialChanged struct {
	UserId int64 `json:"userId"`
	Status bool  `json:"status"`
}

type VideoDialChanges struct {
	ChatId int64               `json:"chatId"`
	Dials  []*VideoDialChanged `json:"dials"`
}

type VideoCallRecordingChangedDto struct {
	RecordInProgress bool  `json:"recordInProgress"`
	ChatId           int64 `json:"chatId"`
}
