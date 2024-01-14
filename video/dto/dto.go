package dto

type VideoInviteDto struct {
	ChatId       int64   `json:"chatId"`
	UserIds      []int64 `json:"userIds"`
	BehalfUserId int64   `json:"behalfUserId"`
	BehalfLogin  string  `json:"behalfLogin"`
}

type VideoCallUserCountChangedDto struct {
	UsersCount int64 `json:"usersCount"`
	ChatId     int64 `json:"chatId"`
}

type VideoCallUsersCallStatusChangedDto struct {
	Users      []VideoCallUserCallStatusChangedDto `json:"users"`
}

// used for drawing red dot - it means user in the call
type VideoCallUserCallStatusChangedDto struct {
	UserId      int64 `json:"userId"`
	IsInVideo   bool  `json:"isInVideo"`
}

type VideoCallScreenShareChangedDto struct {
	ChatId     int64 `json:"chatId"`
	HasScreenShares bool `json:"hasScreenShares"`
}

type VideoCallRecordingChangedDto struct {
	RecordInProgress bool  `json:"recordInProgress"`
	ChatId           int64 `json:"chatId"`
}

type ParticipantBelongsToChat struct {
	UserId  int64 `json:"userId"`
	Belongs bool  `json:"belongs"`
}

type ParticipantsBelongToChat struct {
	Users []*ParticipantBelongsToChat `json:"users"`
}

type ChatName struct {
	Name   string `json:"name"`   // chatName or userName in case tet-a-tet
	UserId int64  `json:"userId"` // userId chatName for
}

// for new call participant
type VideoCallInvitation struct {
	ChatId   int64  `json:"chatId"`
	ChatName string `json:"chatName"`
	Status   string `json:"status"`
}

// for call owner
type VideoDialChanged struct {
	UserId int64 `json:"userId"`
	Status string  `json:"status"`
}

type VideoDialChanges struct {
	ChatId int64               `json:"chatId"`
	Dials  []*VideoDialChanged `json:"dials"`
}

type S3Response struct {
	AccessKey string            `json:"accessKey"`
	Secret    string            `json:"secret"`
	Region    string            `json:"region"`
	Endpoint  string            `json:"endpoint"`
	Bucket    string            `json:"bucket"`
	Metadata  map[string]string `json:"metadata"`
	Filepath  string            `json:"filepath"`
}

type BasicChatDto struct {
	TetATet        bool    `json:"tetATet"`
	ParticipantIds []int64 `json:"participantIds"`
}
