package dto

// input Dto
type StoreNotifyDto struct {
	PeerId    string `json:"peerId"`
	StreamId  string `json:"streamId"`
	Login     string `json:"login"`
	VideoMute bool   `json:"videoMute"`
	AudioMute bool   `json:"audioMute"`
}
