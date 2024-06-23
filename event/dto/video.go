package dto

type VideoCallUserCountChangedDto struct {
	UsersCount int64 `json:"usersCount"`
	ChatId     int64 `json:"chatId"`
	HasScreenShares *bool `json:"hasScreenShares"`
}

type VideoCallScreenShareChangedDto struct {
	ChatId     int64 `json:"chatId"`
	HasScreenShares bool `json:"hasScreenShares"`
}

type VideoInviteDto struct {
	ChatId       int64   `json:"chatId"`
	UserIds      []int64 `json:"userIds"`
	BehalfUserId int64   `json:"behalfUserId"`
	BehalfLogin  string  `json:"behalfLogin"`
}

type VideoCallInvitation struct {
	ChatId   int64   `json:"chatId"`
	ChatName string  `json:"chatName"`
	Status   string  `json:"status"`
	Avatar   *string `json:"avatar"`
}

type VideoDialChanged struct {
	UserId int64 `json:"userId"`
	Status string  `json:"status"`
}

type VideoDialChanges struct {
	ChatId int64               `json:"chatId"`
	Dials  []*VideoDialChanged `json:"dials"`
}

type VideoCallRecordingChangedDto struct {
	RecordInProgress bool  `json:"recordInProgress"`
	ChatId           int64 `json:"chatId"`
}

type VideoCallUsersCallStatusChangedDto struct {
	Users      []VideoCallUserCallStatusChangedDto `json:"users"`
}

type VideoCallUserCallStatusChangedDto struct {
	UserId      int64 `json:"userId"`
	IsInVideo   bool  `json:"isInVideo"`
}
