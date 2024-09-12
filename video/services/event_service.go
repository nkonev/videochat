package services

import (
	"context"
	"github.com/livekit/protocol/livekit"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	"nkonev.name/video/db"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/utils"
)

type StateChangedEventService struct {
	conf                *config.ExtendedConfig
	livekitRoomClient   client.LivekitRoomClient
	userService         *UserService
	notificationService *NotificationService
	egressService       *EgressService
	restClient          *client.RestClient
	rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher
	rabbitMqInvitePublisher *producer.RabbitInvitePublisher
	dialStatusPublisher   *producer.RabbitDialStatusPublisher
}

func NewStateChangedEventService(
	conf *config.ExtendedConfig,
	livekitRoomClient client.LivekitRoomClient,
	userService *UserService,
	notificationService *NotificationService,
	egressService *EgressService,
	restClient *client.RestClient,
	rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher,
	rabbitMqInvitePublisher *producer.RabbitInvitePublisher,
	dialStatusPublisher   *producer.RabbitDialStatusPublisher,
) *StateChangedEventService {
	return &StateChangedEventService{
		conf: conf,
		livekitRoomClient: livekitRoomClient,
		userService: userService,
		notificationService: notificationService,
		egressService: egressService,
		restClient: restClient,
		rabbitUserIdsPublisher: rabbitUserIdsPublisher,
		rabbitMqInvitePublisher: rabbitMqInvitePublisher,
		dialStatusPublisher: dialStatusPublisher,
	}
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
func (h *StateChangedEventService) NotifyAllChatsAboutUsersInVideoStatus(ctx context.Context, tx *db.Tx, userIdsToFilter []int64) {
	if len (userIdsToFilter) > 0 {
		batchUserStates, err := tx.GetUserStatesFiltered(userIdsToFilter)
		if err != nil {
			GetLogEntry(ctx).Errorf("error during reading user states %v", err)
			return
		}
		h.processBatch(ctx, batchUserStates)
	} else {
		offset := int64(0)
		hasMoreElements := true
		for hasMoreElements {
			batchUserStates, err := tx.GetAllUserStates(utils.DefaultSize, offset)
			if err != nil {
				GetLogEntry(ctx).Errorf("error during reading user states %v", err)
				continue
			}
			h.processBatch(ctx, batchUserStates)

			hasMoreElements = len(batchUserStates) == utils.DefaultSize
			offset += utils.DefaultSize
		}
	}
}

func (h *StateChangedEventService) processBatch(ctx context.Context, batchUserStates []dto.UserCallState) {
	var dtos []dto.VideoCallUserCallStatusChangedDto = make([]dto.VideoCallUserCallStatusChangedDto, 0)

	var byUserId = map[int64][]dto.UserCallState{}
	for _, st := range batchUserStates {
		byUserId[st.UserId] = append(byUserId[st.UserId], st)
	}

	for userId, userStates := range byUserId {
		if len(userStates) == 0 {
			continue
		}

		// a situation when user has inCall and removing states simultaneously is _im_possible
		// kinda deduplication (we are looping over the same user)
		userState := userStates[0]

		isInVideo := userState.Status == db.CallStatusInCall
		dtos = append(dtos, dto.VideoCallUserCallStatusChangedDto{
			UserId:    		userId,
			IsInVideo:     	isInVideo,
		})
	}
	// red dot
	err := h.rabbitUserIdsPublisher.Publish(ctx, &dto.VideoCallUsersCallStatusChangedDto{Users: dtos})
	if err != nil {
		GetLogEntry(ctx).Errorf("error during publishing: %v", err)
		return
	}
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
		if ownerId == anUserId {
			continue // not to send invitations to myself
		}

		invitation := dto.VideoCallInvitation{
			ChatId:   chatId,
			Status:   aStatus,
		}

		// this is sending call invitations to all the ivitees
		for _, chatInviteName := range inviteNames {
			if anUserId == chatInviteName.UserId { // we found match between a target userId and chatInviteName for him
				invitation.ChatName = chatInviteName.Name
				invitation.Avatar = GetAvatar(ownerAvatar, tetATet)
				break
			}
		}

		err := h.rabbitMqInvitePublisher.Publish(ctx, &invitation, anUserId)
		if err != nil {
			GetLogEntry(ctx).Error(err, "Error during sending VideoInviteDto")
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

func (h *StateChangedEventService) SendDialEvents(c context.Context, chatId int64, userIdAndStatus map[int64]string, ownerId int64, ownerAvatar string, tetATet bool, inviteNames []*dto.ChatName) {
	GetLogEntry(c).Infof("Sending dial events for %v with ownerId %v", userIdAndStatus, ownerId)

	// updates for ChatParticipants (blinking green tube) and ChatView (blinking it tet-a-tet)
	h.dialStatusPublisher.Publish(c, chatId, userIdAndStatus, ownerId)

	// send the new status (= invitation) immediately to users (callees)
	h.SendInvitationsWithStatuses(c, chatId, ownerId, userIdAndStatus, inviteNames, ownerAvatar, tetATet)
}
