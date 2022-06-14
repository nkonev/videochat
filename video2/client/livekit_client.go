package client

import (
	lksdk "github.com/livekit/server-sdk-go"
	"nkonev.name/video/config"
)

func NewLivekitClient(conf *config.ExtendedConfig) *lksdk.RoomServiceClient {
	client := lksdk.NewRoomServiceClient(conf.LivekitConfig.Url, conf.LivekitConfig.Api.Key, conf.LivekitConfig.Api.Secret)
	return client
}
