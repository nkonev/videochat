package dto

type NotifyRequest struct {
	PeerId    string `json:"peerId"`
	Login     string `json:"login"`
	VideoMute bool   `json:"videoMute"`
	AudioMute bool   `json:"audioMute"`
}
type ChatNotifyDto struct {
	Data       *NotifyRequest `json:"data"`
	UsersCount int64          `json:"usersCount"`
	ChatId     int64          `json:"chatId"`
}
