package client

import (
	lksdk "github.com/livekit/server-sdk-go"
	"nkonev.name/video/config"
)

func NewLivekitClient(conf *config.ExtendedConfig) *lksdk.RoomServiceClient {
	client := lksdk.NewRoomServiceClient(conf.LivekitConfig.Url, conf.LivekitConfig.Api.Key, conf.LivekitConfig.Api.Secret)
	return client
}

func NewEgressClient(conf *config.ExtendedConfig) *lksdk.EgressClient {
	return lksdk.NewEgressClient("http://localhost:7880", "APIznJxWShGW3Kt", "KEUUtCDVRqXk9me0Ok94g8G9xwtnjMeUxfNMy8dow6iA")
}
