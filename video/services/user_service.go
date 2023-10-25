package services

import (
	"context"
	"encoding/json"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/utils"
)

type UserService struct {
	livekitRoomClient *lksdk.RoomServiceClient
}

func NewUserService(livekitRoomClient *lksdk.RoomServiceClient) *UserService {
	return &UserService{
		livekitRoomClient: livekitRoomClient,
	}
}

func (h *UserService) CountUsers(ctx context.Context, roomName string) (int64, bool, error) {
	var req *livekit.ListParticipantsRequest = &livekit.ListParticipantsRequest{Room: roomName}
	participants, err := h.livekitRoomClient.ListParticipants(ctx, req)
	if err != nil {
		return 0, false, err
	}

	var hasScreenShares = false
	for _, p := range participants.Participants {
		for _, t := range p.Tracks {
			if t.Source == livekit.TrackSource_SCREEN_SHARE {
				hasScreenShares = true
				break
			}
		}
		if hasScreenShares {
			break
		}
	}

	var usersCount = int64(len(participants.Participants))
	return usersCount, hasScreenShares, nil
}

func (vh *UserService) GetVideoParticipants(chatId int64, ctx context.Context) ([]int64, error) {
	roomName := utils.GetRoomNameFromId(chatId)

	var ret = []int64{}
	var set = make(map[int64]bool)

	lpr := &livekit.ListParticipantsRequest{Room: roomName}
	participants, err := vh.livekitRoomClient.ListParticipants(ctx, lpr)
	if err != nil {
		Logger.Errorf("Unable to get participants %v", err)
		return ret, err
	}

	for _, participant := range participants.Participants {
		if utils.IsNotHumanUser(participant.Identity) {
			continue
		}

		md := &dto.MetadataDto{}
		err = json.Unmarshal([]byte(participant.Metadata), md)
		if err != nil {
			Logger.Errorf("got error during parsing metadata from chatId=%v, %v", chatId, err)
			continue
		}
		set[md.UserId] = true
	}

	for key, value := range set {
		if value {
			ret = append(ret, key)
		}
	}

	return ret, nil
}
