package client

import (
	"context"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"nkonev.name/video/config"
)

// primarily for testing purposes
//go:generate mockery --name LivekitRoomClient
type LivekitRoomClient interface {
	ListParticipants(ctx context.Context, req *livekit.ListParticipantsRequest) (*livekit.ListParticipantsResponse, error)
	MutePublishedTrack(ctx context.Context, req *livekit.MuteRoomTrackRequest) (*livekit.MuteRoomTrackResponse, error)
	ListRooms(ctx context.Context, req *livekit.ListRoomsRequest) (*livekit.ListRoomsResponse, error)
	RemoveParticipant(ctx context.Context, req *livekit.RoomParticipantIdentity) (*livekit.RemoveParticipantResponse, error)
}

func NewLivekitClient(conf *config.ExtendedConfig) LivekitRoomClient {
	client := lksdk.NewRoomServiceClient(conf.LivekitConfig.Url, conf.LivekitConfig.Api.Key, conf.LivekitConfig.Api.Secret)
	return client
}

func NewEgressClient(conf *config.ExtendedConfig) *lksdk.EgressClient {
	return lksdk.NewEgressClient(conf.LivekitConfig.Url, conf.LivekitConfig.Api.Key, conf.LivekitConfig.Api.Secret)
}
