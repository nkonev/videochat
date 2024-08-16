package services

import (
	"context"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/utils"
)

type StateChangedEventService struct {
	conf                *config.ExtendedConfig
	livekitRoomClient   *lksdk.RoomServiceClient
	userService         *UserService
	notificationService *NotificationService
	egressService       *EgressService
	restClient          *client.RestClient
	redisService        *DialRedisRepository
	rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher
	rabbitMqInvitePublisher *producer.RabbitInvitePublisher
}

func NewStateChangedEventService(conf *config.ExtendedConfig, livekitRoomClient *lksdk.RoomServiceClient, userService *UserService, notificationService *NotificationService, egressService *EgressService, restClient *client.RestClient, redisService *DialRedisRepository, rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher, rabbitMqInvitePublisher *producer.RabbitInvitePublisher) *StateChangedEventService {
	return &StateChangedEventService{conf: conf, livekitRoomClient: livekitRoomClient, userService: userService, notificationService: notificationService, egressService: egressService, restClient: restClient, redisService: redisService, rabbitUserIdsPublisher: rabbitUserIdsPublisher, rabbitMqInvitePublisher: rabbitMqInvitePublisher}
}

func (h *StateChangedEventService) NotifyAllChatsAboutVideoCallUsersCount(ctx context.Context) {
	listRoomReq := &livekit.ListRoomsRequest{}
	rooms, err := h.livekitRoomClient.ListRooms(ctx, listRoomReq)
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

		// Here room.NumParticipants are zeroed, so we need to invoke service
		usersCount, hasScreenShares, err := h.userService.CountUsers(ctx, room.Name)
		if err != nil {
			GetLogEntry(ctx).Errorf("got error during counting users in scheduler, %v", err)
			continue
		}

		err = h.restClient.GetChatParticipantIds(ctx, chatId, func(participantIds []int64) error {
			GetLogEntry(ctx).Debugf("Sending user count in video changed chatId=%v, usersCount=%v", chatId, usersCount)
			internalErr := h.notificationService.NotifyVideoUserCountChanged(ctx, participantIds, chatId, usersCount)
			if internalErr != nil {
				GetLogEntry(ctx).Errorf("got error during notificationService.NotifyVideoUserCountChanged, %v", internalErr)
			}

			internalErr = h.notificationService.NotifyVideoScreenShareChanged(ctx, participantIds, chatId, hasScreenShares)
			if internalErr != nil {
				GetLogEntry(ctx).Errorf("got error during notificationService.NotifyVideoScreenShareChanged, %v", internalErr)
			}
			return internalErr
		})
		if err != nil {
			GetLogEntry(ctx).Error(err, "Failed during getting chat participantIds")
			continue
		}
	}
}


// sends info about "red dot"
func (h *StateChangedEventService) NotifyAllChatsAboutUsersVideoStatus(ctx context.Context, userIdsToFilter []int64) {
	userIds, err := h.redisService.GetUserIds(ctx)
	if err != nil {
		GetLogEntry(ctx).Error(err, "error during reading userIds %v", err)
		return
	}

	var dtos []dto.VideoCallUserCallStatusChangedDto = make([]dto.VideoCallUserCallStatusChangedDto, 0)
	for _, userId := range userIds {
		if userIdsToFilter != nil {
			if !utils.Contains(userIdsToFilter, userId) {
				continue
			}
		}

		status, err := h.redisService.GetUserCallStatus(ctx, userId)
		if err != nil {
			GetLogEntry(ctx).Error(err, "error during reading userStatus, userId = %v", userId, err)
			continue
		}
		isInVideo := status == CallStatusInCall
		dtos = append(dtos, dto.VideoCallUserCallStatusChangedDto{
			UserId:    		userId,
			IsInVideo:     	isInVideo,
		})
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during notifying about user is in video, userId=%v, error=%v", userId, err)
			continue
		}
	}
	err = h.rabbitUserIdsPublisher.Publish(ctx, &dto.VideoCallUsersCallStatusChangedDto{Users: dtos})
}

func (h *StateChangedEventService) NotifyAllChatsAboutVideoCallRecording(ctx context.Context) {
	listRoomReq := &livekit.ListRoomsRequest{}
	rooms, err := h.livekitRoomClient.ListRooms(ctx, listRoomReq)
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

		activeEgresses, err := h.egressService.GetActiveEgresses(ctx, chatId)
		if err != nil {
			GetLogEntry(ctx).Errorf("got error during counting active egresses in scheduler, %v", err)
			continue
		}

		var recordInProgressByOwner = make(map[int64]bool)
		for _, ownerId := range activeEgresses {
			recordInProgressByOwner[ownerId] = true
		}

		err = h.notificationService.NotifyRecordingChanged(ctx, chatId, recordInProgressByOwner)
		if err != nil {
			GetLogEntry(ctx).Errorf("got error during notificationService.NotifyRecordingChanged, %v", err)
		}

	}
}

// sends invitations "smb called you to chat x"
func (h *StateChangedEventService) SendInvitationsWithStatuses(ctx context.Context, chatId, ownerId int64, statuses map[int64]string, inviteNames []*dto.ChatName, ownerAvatar string, tetATet bool) {
	if len(statuses) == 0 {
		return
	}

	for anUserId, aStatus := range statuses {
		// this is sending call invitations to all the ivitees
		for _, chatInviteName := range inviteNames {
			if ownerId == chatInviteName.UserId {
				continue // not to send invitations to myself
			}

			if anUserId == chatInviteName.UserId {

				invitation := dto.VideoCallInvitation{
					ChatId:   chatId,
					ChatName: chatInviteName.Name,
					Status:   aStatus,
				}

				invitation.Avatar = GetAvatar(ownerAvatar, tetATet)

				err := h.rabbitMqInvitePublisher.Publish(ctx, &invitation, chatInviteName.UserId)
				if err != nil {
					GetLogEntry(ctx).Error(err, "Error during sending VideoInviteDto")
				}
			}
		}
	}
}

func GetAvatar(ownerAvatar string, tetATet bool) *string {
	if tetATet {
		return &ownerAvatar
	} else {
		return nil
	}
}
