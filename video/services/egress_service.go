package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	. "nkonev.name/video/logger"
	"nkonev.name/video/utils"
)

const ownerIdMetadataKey = "ownerid"

type EgressService struct {
	egressClient *lksdk.EgressClient
}

func NewEgressService(egressClient *lksdk.EgressClient) *EgressService {
	return &EgressService{egressClient: egressClient}
}

func (rh *EgressService) GetActiveEgresses(chatId int64, ctx context.Context) (map[string]int64, error) {
	aRoomId := utils.GetRoomNameFromId(chatId)

	listRequest := livekit.ListEgressRequest{
		RoomName: aRoomId,
	}
	egresses, err := rh.egressClient.ListEgress(ctx, &listRequest)
	if err != nil {
		GetLogEntry(ctx).Errorf("Unable to get egresses")
		return nil, errors.New("Unable to get egresses")
	}

	ret := map[string]int64{}
	for _, egress := range egresses.Items {
		if egress.Status == livekit.EgressStatus_EGRESS_ACTIVE && egress.EndedAt == 0 {
			ownerId, err := rh.GetOwnerId(ctx, egress)
			if err != nil {
				GetLogEntry(ctx).Errorf("Unable to get ownerId of %v: %v", egress.EgressId, err)
			} else {
				ret[egress.EgressId] = ownerId
			}
		}
	}

	return ret, nil
}

func (rh *EgressService) GetOwnerId(ctx context.Context, egress *livekit.EgressInfo) (int64, error) {
	var ownerId int64
	wasSet := false
	inf := egress.Request
	ic, ok := inf.(*livekit.EgressInfo_RoomComposite)
	if ok {
		fileOutputs := ic.RoomComposite.FileOutputs
		if len(fileOutputs) > 0 {
			fileOutput := fileOutputs[0]
			aS3 := fileOutput.GetS3()
			if aS3 != nil {
				ownerIdString, ok := aS3.Metadata[ownerIdMetadataKey]
				if ok {
					anOwnerId, err := utils.ParseInt64(ownerIdString)
					if err != nil {
						GetLogEntry(ctx).Errorf("Unable to parse owner id: %v", err)
					} else {
						ownerId = anOwnerId
						wasSet = true
					}
				}
			}
		}
	}
	if !wasSet {
		return 0, fmt.Errorf("Unable to get owner id for egress %v", egress.EgressId)
	}
	return ownerId, nil
}
