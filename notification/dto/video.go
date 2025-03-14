package dto

// for new call participant
type VideoCallInvitation struct {
	ChatId   int64   `json:"chatId"`
	ChatName string  `json:"chatName"`
	Status   string  `json:"status"`
	Avatar   *string `json:"avatar"`
}
