package services

import (
	"context"
	"github.com/livekit/protocol/livekit"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/video/client"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/utils"
)

type UserService struct {
	livekitRoomClient client.LivekitRoomClient
	tr trace.Tracer
}

func NewUserService(livekitRoomClient client.LivekitRoomClient) *UserService {
	tr := otel.Tracer("userService")

	return &UserService{
		livekitRoomClient: livekitRoomClient,
		tr: tr,
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

func (vh *UserService) GetVideoParticipants(ctx context.Context, chatId int64) ([]dto.UserCallStateId, error) {
	roomName := utils.GetRoomNameFromId(chatId)

	var ret = []dto.UserCallStateId{}
	var set = make(map[dto.UserCallStateId]bool)

	lpr := &livekit.ListParticipantsRequest{Room: roomName}
	participants, err := vh.livekitRoomClient.ListParticipants(ctx, lpr)
	if err != nil {
		GetLogEntry(ctx).Errorf("Unable to get participants %v", err)
		return ret, err
	}

	for _, participant := range participants.Participants {
		metadata, err := utils.ParseParticipantMetadataOrNull(participant)
		if err != nil {
			GetLogEntry(ctx).Errorf("got error during parsing metadata from participant=%v chatId=%v, %v", participant, chatId, err)
			continue
		}
		if metadata == nil {
			continue
		}
		set[dto.UserCallStateId{
			UserId: metadata.UserId,
			TokenId: metadata.TokenId,
		}] = true
	}

	for key, value := range set {
		if value {
			ret = append(ret, key)
		}
	}

	return ret, nil
}

func (vh *UserService) KickUserHavingChatId(ctx context.Context, chatId, userId int64) {
	roomName := utils.GetRoomNameFromId(chatId)

	lpr := &livekit.ListParticipantsRequest{Room: roomName}
	participants, err := vh.livekitRoomClient.ListParticipants(ctx, lpr)
	if err != nil {
		GetLogEntry(ctx).Errorf("Unable to get participants %v", err)
		return
	}

	for _, participant := range participants.Participants {
		metadata, err := utils.ParseParticipantMetadataOrNull(participant)
		if err != nil {
			GetLogEntry(ctx).Errorf("got error during parsing metadata from participant=%v chatId=%v, %v", participant, chatId, err)
			continue
		}
		if metadata == nil {
			continue
		}
		if metadata.UserId == userId {
			var removeReq = &livekit.RoomParticipantIdentity{
				Room:     roomName,
				Identity: participant.Identity,
			}
			GetLogEntry(ctx).Infof("Kicking userId=%v with identity %v from chatId=%v", userId, participant.Identity, chatId)
			_, err := vh.livekitRoomClient.RemoveParticipant(ctx, removeReq)
			if err != nil {
				GetLogEntry(ctx).Errorf("got error during kicking userId=%v, %v", userId, err)
				continue
			}
		}
	}
}

func (vh *UserService) KickUser(ctx context.Context, userId int64) {
	ctx, span := vh.tr.Start(ctx, "user.kick")
	defer span.End()
	span.SetAttributes(attribute.Int64("userId", userId))

	listRoomReq := &livekit.ListRoomsRequest{}
	rooms, err := vh.livekitRoomClient.ListRooms(ctx, listRoomReq)
	if err != nil {
		GetLogEntry(ctx).Error(err, "error during reading rooms %v", err)
		return
	}

	for _, room := range rooms.Rooms {
		chatId, err := utils.GetRoomIdFromName(room.Name)
		if err != nil {
			GetLogEntry(ctx).Errorf("got error during getting chat id from roomName %v %v", room.Name, err)
			continue
		}

		lpr := &livekit.ListParticipantsRequest{Room: room.Name}
		participants, err := vh.livekitRoomClient.ListParticipants(ctx, lpr)
		if err != nil {
			GetLogEntry(ctx).Errorf("Unable to get participants %v", err)
			continue
		}

		for _, participant := range participants.Participants {
			metadata, err := utils.ParseParticipantMetadataOrNull(participant)
			if err != nil {
				GetLogEntry(ctx).Errorf("got error during parsing metadata from participant=%v chatId=%v, %v", participant, chatId, err)
				continue
			}
			if metadata == nil {
				continue
			}
			if metadata.UserId == userId {
				var removeReq = &livekit.RoomParticipantIdentity{
					Room:     room.Name,
					Identity: participant.Identity,
				}
				GetLogEntry(ctx).Infof("Kicking userId=%v with identity %v from chatId=%v", userId, participant.Identity, chatId)
				_, err := vh.livekitRoomClient.RemoveParticipant(ctx, removeReq)
				if err != nil {
					GetLogEntry(ctx).Errorf("got error during kicking userId=%v, %v", userId, err)
					continue
				}
			}
		}
	}
}
