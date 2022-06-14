package dto

// stored in video and used for notifications
type NotifyDto struct {
	UserId    int64  `json:"userId"`
	Login     string `json:"login"`
	Avatar    string `json:"avatar"`
	VideoMute bool   `json:"videoMute"`
	AudioMute bool   `json:"audioMute"`
}
