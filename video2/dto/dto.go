package dto

type MuteInfo struct {
	Kind  string `json:"kind"`
	Muted bool   `json:"muted"`
}

// used for notifications
type NotifyDto struct {
	UserId      int64               `json:"userId"`
	Login       string              `json:"login"`
	MutedTracks map[string]MuteInfo `json:"mutedTracks"`
}
