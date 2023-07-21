package services

import (
	"context"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/utils"
)

type StateChangedEventService struct {
	conf                *config.ExtendedConfig
	livekitRoomClient   *lksdk.RoomServiceClient
	userService         *UserService
	notificationService *NotificationService
	egressService       *EgressService
	restClient          *client.RestClient
}

func NewStateChangedEventService(conf *config.ExtendedConfig, livekitRoomClient *lksdk.RoomServiceClient, userService *UserService, notificationService *NotificationService, egressService *EgressService, restClient *client.RestClient) *StateChangedEventService {
	return &StateChangedEventService{conf: conf, livekitRoomClient: livekitRoomClient, userService: userService, notificationService: notificationService, egressService: egressService, restClient: restClient}
}

func (h *StateChangedEventService) NotifyAllChatsAboutVideoCallUsersCount(ctx context.Context) {
	listRoomReq := &livekit.ListRoomsRequest{}
	rooms, err := h.livekitRoomClient.ListRooms(context.Background(), listRoomReq)
	if err != nil {
		Logger.Error(err, "error during reading rooms %v", err)
		return
	}
	for _, room := range rooms.Rooms {
		chatId, err := utils.GetRoomIdFromName(room.Name)
		if err != nil {
			Logger.Errorf("got error during getting chat id from roomName %v %v", room.Name, err)
			continue
		}

		// Here room.NumParticipants are zeroed, so we need to invoke service
		usersCount, hasScreenShares, err := h.userService.CountUsers(context.Background(), room.Name)
		if err != nil {
			Logger.Errorf("got error during counting users in scheduler, %v", err)
		} else {
			participantIds, err := h.restClient.GetChatParticipantIds(chatId, ctx)
			if err != nil {
				Logger.Error(err, "Failed during getting chat participantIds")
				return
			}

			Logger.Debugf("Sending user count in video changed chatId=%v, usersCount=%v", chatId, usersCount)
			err = h.notificationService.NotifyVideoUserCountChanged(participantIds, chatId, usersCount, ctx)
			if err != nil {
				Logger.Errorf("got error during notificationService.NotifyVideoUserCountChanged, %v", err)
			}

			err = h.notificationService.NotifyVideoScreenShareChanged(participantIds, chatId, hasScreenShares, ctx)
			if err != nil {
				Logger.Errorf("got error during notificationService.NotifyVideoScreenShareChanged, %v", err)
			}
		}
	}
}

func (h *StateChangedEventService) NotifyAllChatsAboutVideoCallRecording(ctx context.Context) {
	listRoomReq := &livekit.ListRoomsRequest{}
	rooms, err := h.livekitRoomClient.ListRooms(context.Background(), listRoomReq)
	if err != nil {
		Logger.Error(err, "error during reading rooms %v", err)
		return
	}
	for _, room := range rooms.Rooms {
		chatId, err := utils.GetRoomIdFromName(room.Name)
		if err != nil {
			Logger.Errorf("got error during getting chat id from roomName %v %v", room.Name, err)
			continue
		}

		recordInProgress, err := h.egressService.HasActiveEgresses(chatId, ctx)
		if err != nil {
			Logger.Errorf("got error during counting active egresses in scheduler, %v", err)
		} else {
			Logger.Debugf("Sending recording changed chatId=%v, recordInProgress=%v", chatId, recordInProgress)
			err = h.notificationService.NotifyRecordingChanged(chatId, recordInProgress, ctx)
			if err != nil {
				Logger.Errorf("got error during notificationService.NotifyRecordingChanged, %v", err)
			}
		}
	}
}
