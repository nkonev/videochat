package services

import (
	"context"
	"errors"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	. "nkonev.name/video/logger"
	"nkonev.name/video/utils"
)

type EgressService struct {
	egressClient *lksdk.EgressClient
}

func NewEgressService(egressClient *lksdk.EgressClient) *EgressService {
	return &EgressService{egressClient: egressClient}
}

func (rh *EgressService) GetActiveEgresses(chatId int64, ctx context.Context) ([]string, error) {
	aRoomId := utils.GetRoomNameFromId(chatId)

	listRequest := livekit.ListEgressRequest{
		RoomName: aRoomId,
	}
	egresses, err := rh.egressClient.ListEgress(ctx, &listRequest)
	if err != nil {
		GetLogEntry(ctx).Errorf("Unable to get egresses")
		return nil, errors.New("Unable to get egresses")
	}

	ret := []string{}
	for _, egress := range egresses.Items {
		if egress.Status == livekit.EgressStatus_EGRESS_ACTIVE && egress.EndedAt == 0 {
			ret = append(ret, egress.EgressId)
		}
	}

	return ret, nil
}

func (rh *EgressService) HasActiveEgresses(chatId int64, ctx context.Context) (bool, error) {
	egresses, err := rh.GetActiveEgresses(chatId, ctx)
	if err != nil {
		return false, err
	}

	recordInProgress := len(egresses) > 0

	return recordInProgress, nil
}
