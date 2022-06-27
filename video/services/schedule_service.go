package services

import (
	"context"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/utils"
)

type ScheduledService struct {
	conf                *config.ExtendedConfig
	livekitRoomClient   *lksdk.RoomServiceClient
	userService         *UserService
	notificationService *NotificationService
}

func NewScheduledService(conf *config.ExtendedConfig, livekitRoomClient *lksdk.RoomServiceClient, userService *UserService, notificationService *NotificationService) *ScheduledService {
	return &ScheduledService{conf: conf, livekitRoomClient: livekitRoomClient, userService: userService, notificationService: notificationService}
}

func (h *ScheduledService) NotifyAllChats() {
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
		usersCount, err := h.userService.CountUsers(context.Background(), room.Name)
		if err != nil {
			Logger.Errorf("got error during counting users in scheduler, %v", err)
			continue
		}
		Logger.Infof("Sending notificationDto chatId=%v", chatId)
		err = h.notificationService.Notify(chatId, usersCount, nil)
		if err != nil {
			Logger.Errorf("got error during notificationService.Notify, %v", err)
			continue
		}
	}
}
