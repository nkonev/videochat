package dto

// input Dto
type UserInputDto struct {
	Avatar    string `json:"avatar"`
	PeerId    string `json:"peerId"`
	StreamId  string `json:"streamId"`
	VideoMute bool   `json:"videoMute"`
	AudioMute bool   `json:"audioMute"`
}

// stored in video and used for notifications
type StoreNotifyDto struct {
	UserId    int64  `json:"userId"`
	StreamId  string `json:"streamId"`
	Login     string `json:"login"`
	Avatar    string `json:"avatar"`
	VideoMute bool   `json:"videoMute"`
	AudioMute bool   `json:"audioMute"`
}

type AaaUserDto struct {
	Id     int64  `json:"id"`
	Login  string `json:"login"`
	Avatar string `json:"avatar"`
}
