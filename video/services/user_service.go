package services

import (
	"context"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
)

type UserService struct {
	livekitRoomClient *lksdk.RoomServiceClient
}

func NewUserService(livekitRoomClient *lksdk.RoomServiceClient) *UserService {
	return &UserService{
		livekitRoomClient: livekitRoomClient,
	}
}

func (h *UserService) CountUsers(ctx context.Context, roomName string) (int64, error) {
	var req *livekit.ListParticipantsRequest = &livekit.ListParticipantsRequest{Room: roomName}
	participants, err := h.livekitRoomClient.ListParticipants(ctx, req)
	if err != nil {
		return 0, err
	}

	var usersCount = int64(len(participants.Participants))
	return usersCount, nil
}
