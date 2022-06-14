package dto

type MuteInfo struct {
	Kind  string `json:"kind"`
	Muted bool   `json:"muted"`
}

type NotifyRequest struct {
	UserId      int64               `json:"userId"`
	Login       string              `json:"login"`
	MutedTracks map[string]MuteInfo `json:"mutedTracks"`
}
type ChatNotifyDto struct {
	Data       *NotifyRequest `json:"data"`
	UsersCount int64          `json:"usersCount"`
	ChatId     int64          `json:"chatId"`
}
