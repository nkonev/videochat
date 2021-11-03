package dto

// input Dto
type StoreNotifyDto struct {
	UserId    int64 `json:"userId"`
	StreamId  string `json:"streamId"`
	Login     string `json:"login"`
	VideoMute bool   `json:"videoMute"`
	AudioMute bool   `json:"audioMute"`
}
