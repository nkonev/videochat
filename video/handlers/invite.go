package handlers

import (
	"context"
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	"nkonev.name/video/dto"
	"nkonev.name/video/logger"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
)

type InviteHandler struct {
	dialRedisRepository   *services.DialRedisRepository
	chatClient            *client.RestClient
	dialStatusPublisher   *producer.RabbitDialStatusPublisher
	notificationPublisher *producer.RabbitNotificationsPublisher
	userService           *services.UserService
	stateChangedEventService *services.StateChangedEventService
}

const EventMissedCall = "missed_call"

func NewInviteHandler(
	dialService *services.DialRedisRepository,
	chatClient *client.RestClient,
	dialStatusPublisher *producer.RabbitDialStatusPublisher,
	notificationPublisher *producer.RabbitNotificationsPublisher,
	userService *services.UserService,
	stateChangedEventService *services.StateChangedEventService,
) *InviteHandler {
	return &InviteHandler{
		dialRedisRepository:   dialService,
		chatClient:            chatClient,
		dialStatusPublisher:   dialStatusPublisher,
		notificationPublisher: notificationPublisher,
		userService:           userService,
		stateChangedEventService: stateChangedEventService,
	}
}

// used by owner to add or remove from dial list
func (vh *InviteHandler) ProcessCallInvitation(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		Logger.Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	callee, err := utils.ParseInt64(c.QueryParam("userId"))
	if err != nil {
		return err
	}

	addToCall, err := utils.ParseBoolean(c.QueryParam("call"))
	if err != nil {
		return err
	}

	// check my access to chat
	if ok, err := vh.chatClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	ok, code := vh.checkAccessOfAnotherUser(c.Request().Context(), callee, chatId, userPrincipalDto)
	if !ok {
		return c.NoContent(code)
	}

	basicChatInfo, err := vh.chatClient.GetBasicChatInfo(chatId, userPrincipalDto.UserId, c.Request().Context()) // tet-a-tet
	if err != nil {
		return err
	}

	if addToCall {
		return c.NoContent(vh.addToCalling(c, callee, chatId, userPrincipalDto, basicChatInfo.TetATet))
	} else {
		return c.NoContent(vh.removeFromCalling(c, callee, chatId, userPrincipalDto))
	}
}

func (vh *InviteHandler) checkAccessOfAnotherUser(ctx context.Context, callee int64, chatId int64, userPrincipalDto *auth.AuthResult) (bool, int) {
	// check participant's access to chat
	if ok, err := vh.chatClient.CheckAccess(callee, chatId, ctx); err != nil {
		return false, http.StatusInternalServerError
	} else if !ok {
		return false, http.StatusUnauthorized
	}

	return true, http.StatusOK
}

func (vh *InviteHandler) addToCalling(c echo.Context, callee int64, chatId int64, userPrincipalDto *auth.AuthResult, tetATet bool) int {
	status, err := vh.dialRedisRepository.GetUserCallStatus(c.Request().Context(), callee)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}

	if !services.CanOverrideCallStatus(status) {
		return http.StatusConflict
	}

	// we remove callee's previous inviting - only after CanOverrideCallStatus() check
	vh.removePrevious(c, callee, true)

	err = vh.dialRedisRepository.AddToDialList(c.Request().Context(), callee, chatId, userPrincipalDto.UserId, services.CallStatusInviting, userPrincipalDto.Avatar, tetATet)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}

	// for better user experience
	vh.sendEvents(c, chatId, []int64{callee}, services.CallStatusInviting, userPrincipalDto.UserId, userPrincipalDto.Avatar, tetATet)

	return http.StatusOK
}

func (vh *InviteHandler) removeFromCalling(c echo.Context, callee int64, chatId int64, userPrincipalDto *auth.AuthResult) int {

	status, _, _, _, ownerId, ownerAvatar, tetATet, err := vh.dialRedisRepository.GetUserCallState(c.Request().Context(), callee)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error during getting ownerId: %v", err)
		return http.StatusInternalServerError
	}

	missedUsersMapWithPreviousStatus := getMapWithSameStatus([]int64{callee}, status)

	code := vh.removeFromCallingList(c, ownerId, ownerAvatar, tetATet, chatId, []int64{callee}, services.CallStatusRemoving)
	if code != http.StatusOK {
		return code
	}

	// if we remove user from call - send them EventMissedCall notification
	vh.sendMissedCallNotification(c.Request().Context(), chatId, userPrincipalDto, missedUsersMapWithPreviousStatus)

	return http.StatusOK
}

// user enters to call somehow, either by clicking green tube or opening .../video link
func (vh *InviteHandler) ProcessEnterToDial(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		Logger.Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	// check my access to chat
	if ok, err := vh.chatClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	// during entering into dial. Returns status: true which means that frontend should (initially) draw the calling.
	// Now it used only in tet-a-tet.
	// If we are in the tet-a-tet
	basicChatInfo, err := vh.chatClient.GetBasicChatInfo(chatId, userPrincipalDto.UserId, c.Request().Context()) // tet-a-tet
	if err != nil {
		return err
	}

	// first of all we remove our previous inviting
	vh.removePrevious(c, userPrincipalDto.UserId, false)

	maybeStatus, _, _, _, maybeOwnerId, maybeOwnerAvatar, maybeTetATet, err := vh.dialRedisRepository.GetUserCallState(c.Request().Context(), userPrincipalDto.UserId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error during getting ownerId: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	// if videochat does not exist OR user call status can be overridden
	// then it means that I'm the owner and I need to create it
	if maybeOwnerId == services.NoUser || services.CanOverrideCallStatus(maybeStatus) {
		// and put myself with a status "inCall"
		err = vh.dialRedisRepository.AddToDialList(c.Request().Context(), userPrincipalDto.UserId, chatId, userPrincipalDto.UserId, services.CallStatusInCall, userPrincipalDto.Avatar, basicChatInfo.TetATet)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during adding as owner %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		usersOfChat := basicChatInfo.ParticipantIds // here are only first 20 users, but it's enough for sake tet-a-tet purposes

		// in this block we start calling in case valid tet-a-tet
		if basicChatInfo.TetATet && len(usersOfChat) == 2 {
			if err != nil {
				logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
				return c.NoContent(http.StatusInternalServerError)
			}
			var oppositeUser *int64 = utils.GetOppositeUser(usersOfChat, userPrincipalDto.UserId)

			// uniq users by userId
			usersOfVideo, err := vh.userService.GetVideoParticipants(chatId, c.Request().Context())
			if err != nil {
				logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
				return c.NoContent(http.StatusInternalServerError)
			}

			var oppositeUserOfVideo *int64 = utils.GetOppositeUser(usersOfVideo, userPrincipalDto.UserId)

			// oppositeUserOfVideo is need for case when your counterpart enters into call (not entered until this moment) and this (oppositeUserOfVideo == nil) prevents us to start calling him back
			// and we(behalf user) doesn't have incoming call
			if oppositeUserOfVideo == nil && oppositeUser != nil {
				// we should call the counterpart (opposite user)
				vh.addToCalling(c, *oppositeUser, chatId, userPrincipalDto, basicChatInfo.TetATet)
			}
		}
	} else { // we enter to somebody's chat
		err = vh.dialRedisRepository.AddToDialList(c.Request().Context(), userPrincipalDto.UserId, chatId, maybeOwnerId, services.CallStatusInCall, maybeOwnerAvatar, basicChatInfo.TetATet)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during adding as non-owner %v", err)
			return err
		}
		vh.sendEvents(c, chatId, []int64{userPrincipalDto.UserId}, services.CallStatusInCall, maybeOwnerId, maybeOwnerAvatar, maybeTetATet)
	}

	vh.stateChangedEventService.NotifyAllChatsAboutUsersVideoStatus(c.Request().Context())

	return c.NoContent(http.StatusOK)
}

func (vh *InviteHandler) removePrevious(c echo.Context, userId int64, removeUserState bool) {
	previousUserCallState, _, _, _, userCallOwnerId, _, _, err := vh.dialRedisRepository.GetUserCallState(c.Request().Context(), userId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Unable to get user call state %v", err)
	}
	if previousUserCallState != services.CallStatusNotFound {
		vh.dialRedisRepository.RemoveFromDialList(c.Request().Context(), userId, removeUserState, userCallOwnerId)
	}
}


func (vh *InviteHandler) ProcessCancelCall(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		Logger.Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	// check my access to chat
	if ok, err := vh.chatClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	_, _, _, _, ownerId, ownerAvatar, tetATet, err := vh.dialRedisRepository.GetUserCallState(c.Request().Context(), userPrincipalDto.UserId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error during getting ownerId: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(vh.removeFromCallingList(c, ownerId, ownerAvatar, tetATet, chatId, []int64{userPrincipalDto.UserId}, services.CallStatusCancelling))
}

// question: how not to overwhelm the system by iterating over all the users and all the chats ?
// answer: using opened rooms and rooms are going to be closed - see livekit's room.empty_timeout


func (vh *InviteHandler) removeFromCallingList(c echo.Context, ownerId int64, ownerAvatar string, tetATet bool, chatId int64, usersOfDial []int64, callStatus string) int {
	var err error
	// we remove callee by setting status
	for _, userId := range usersOfDial {

		if ownerId == services.NoUser {
			continue
		}

		err = vh.setUserStatus(c.Request().Context(), userId, chatId, callStatus) // here
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		}

		vh.sendEvents(c, chatId, []int64{userId}, callStatus, ownerId, ownerAvatar, tetATet)
	}
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func (vh *InviteHandler) sendEvents(c echo.Context, chatId int64, usersOfDial []int64, callStatus string, ownerId int64, ownerAvatar string, tetATet bool) {
	// we send "stop-inviting-for-userPrincipalDto.UserId-signal" or "start-" to the call's owner, depending on callStatus
	vh.dialStatusPublisher.Publish(c.Request().Context(), chatId, getMapWithSameStatus(usersOfDial, callStatus), ownerId)

	// send the new status immediately to user
	vh.stateChangedEventService.SendInvitationsWithStatuses(c.Request().Context(), chatId, ownerId, getMapWithSameStatus(usersOfDial, callStatus), ownerAvatar, tetATet)
}

func getMapWithSameStatus(userIds []int64, status string) map[int64]string {
	var ret = map[int64]string{}
	for _, userId := range userIds {
		ret[userId] = status
	}
	return ret
}

func(vh *InviteHandler) setUserStatus(ctx context.Context, callee, chatId int64, callStatus string) error {
	err := vh.dialRedisRepository.SetUserStatus(ctx, callee, callStatus)
	if err != nil {
		return err
	}
	if services.ShouldProlong(callStatus) {
		err = vh.dialRedisRepository.ResetExpiration(ctx, callee)
		if err != nil {
			return err
		}
		err = vh.dialRedisRepository.ResetOwner(ctx, callee)
		if err != nil {
			return err
		}
	}
	if services.IsTemporary(callStatus) {
		err = vh.dialRedisRepository.SetCurrentTimeForRemoving(ctx, callee)
		if err != nil {
			return err
		}
	}
	return err
}

// owner stops call by exiting
func (vh *InviteHandler) ProcessLeave(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		Logger.Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	// check my access to chat
	if ok, err := vh.chatClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	_, _, _, _, ownerId, ownerAvatar, tetATet, err := vh.dialRedisRepository.GetUserCallState(c.Request().Context(), userPrincipalDto.UserId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error during getting ownerId: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if ownerId == userPrincipalDto.UserId { // owner leaving

		// callees of me
		redisUsersOfDial, err := vh.dialRedisRepository.GetUserCalls(c.Request().Context(), userPrincipalDto.UserId)
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		videoParticipants, err := vh.userService.GetVideoParticipants(chatId, c.Request().Context())
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		// sometimes in race-condition manner due to livekit the call owner can be here, so we remove them by not to adding
		videoParticipantsNormalized := make([]int64, 0)
		for _, vuid := range videoParticipants {
			if vuid != userPrincipalDto.UserId {
				videoParticipantsNormalized = append(videoParticipantsNormalized, vuid)
			}
		}
		redisUsersNormalized := make([]int64, 0)
		for _, ruid := range redisUsersOfDial {
			if ruid != userPrincipalDto.UserId {
				redisUsersNormalized = append(redisUsersNormalized, ruid)
			}
		}

		missedUsers := make([]int64, 0)
		inVideoUsers := make([]int64, 0)
		for _, redisCallUser := range redisUsersNormalized {
			if !utils.Contains(videoParticipantsNormalized, redisCallUser) {
				missedUsers = append(missedUsers, redisCallUser)
			} else {
				inVideoUsers = append(inVideoUsers, redisCallUser)
			}
		}

		// the owner removes all the dials by setting status
		toRemove := make([]int64, 0) // missed users + myself
		for _, u := range missedUsers {
			toRemove = append(toRemove, u)
		}
		toRemove = append(toRemove, userPrincipalDto.UserId)

		missedUsersMapWithPreviousStatus := vh.getUsersWithStatuses(c, missedUsers)

		vh.removeFromCallingList(c, ownerId, ownerAvatar, tetATet, chatId, toRemove, services.CallStatusRemoving)

		// for all participants to dial - send EventMissedCall notification
		vh.sendMissedCallNotification(c.Request().Context(), chatId, userPrincipalDto, missedUsersMapWithPreviousStatus)

		// delegate ownership to another user
		vh.dialRedisRepository.TransferOwnership(c.Request().Context(), inVideoUsers, userPrincipalDto.UserId, chatId)
	} else {
		// set myself to temporarily status
		vh.removeFromCallingList(c, ownerId, ownerAvatar, tetATet, chatId, []int64{userPrincipalDto.UserId}, services.CallStatusRemoving)
	}
	vh.stateChangedEventService.NotifyAllChatsAboutUsersVideoStatus(c.Request().Context())
	return c.NoContent(http.StatusOK)
}

func (vh *InviteHandler) getUsersWithStatuses(c echo.Context, missedUsers []int64) map[int64]string {
	missedUsersMap := make(map[int64]string)
	for _, mu := range missedUsers {
		status, err := vh.dialRedisRepository.GetUserCallStatus(c.Request().Context(), mu)
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			continue
		}
		missedUsersMap[mu] = status
	}
	return missedUsersMap
}

func (vh *InviteHandler) sendMissedCallNotification(ctx context.Context, chatId int64, userPrincipalDto *auth.AuthResult, missedUsers map[int64]string) {
	missedUsersList := make([]int64, 0)
	for mu, _ := range missedUsers {
		missedUsersList = append(missedUsersList, mu)
	}

	if len(missedUsers) > 0 {
		if chatNames, err := vh.chatClient.GetChatNameForInvite(chatId, userPrincipalDto.UserId, missedUsersList, ctx); err != nil {
			logger.GetLogEntry(ctx).Errorf("Error %v", err)
		} else {
			for _, chatName := range chatNames {

				if !services.IsTemporary(missedUsers[chatName.UserId]) {
					// here send missed call notification
					var missedCall = dto.NotificationEvent{
						EventType:              EventMissedCall,
						ChatId:                 chatId,
						UserId:                 chatName.UserId,
						MissedCallNotification: &dto.MissedCallNotification{chatName.Name},
						ByUserId:               userPrincipalDto.UserId,
						ByLogin:                userPrincipalDto.UserLogin,
					}
					if len(userPrincipalDto.Avatar) > 0 {
						missedCall.ByAvatar = &userPrincipalDto.Avatar
					}

					err = vh.notificationPublisher.Publish(ctx, missedCall)
					if err != nil {
						logger.GetLogEntry(ctx).Errorf("Error %v", err)
					}
				}
			}
		}
	}
}

// send current dial statuses to WebSocket
func (vh *InviteHandler) SendDialStatusChangedToCallOwner(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		Logger.Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	// check my access to chat
	if ok, err := vh.chatClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	vh.stateChangedEventService.NotifyAllChatsAboutUsersVideoStatus(c.Request().Context())

	return c.NoContent(http.StatusOK)
}

func (vh *InviteHandler) GetInvitationStatus(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		Logger.Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	userCallState, chatId, _, _, userCallOwnerId, userCallOwnerAvatar, tetATet, err := vh.dialRedisRepository.GetUserCallState(c.Request().Context(), userPrincipalDto.UserId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Unable to get user call state %v", err)
	}

	var inviteName string
	if userCallState != services.CallStatusNotFound {
		inviteNames, err := vh.chatClient.GetChatNameForInvite(chatId, userCallOwnerId, []int64{userPrincipalDto.UserId}, c.Request().Context())
		if err != nil {
			GetLogEntry(c.Request().Context()).Error(err, "Failed during getting chat invite names")
			return c.NoContent(http.StatusInternalServerError)
		}

		if len(inviteNames) != 1 {
			return c.NoContent(http.StatusNoContent)
		}
		chatInviteName := inviteNames[0]
		inviteName = chatInviteName.Name
	}


	invitation := dto.VideoCallInvitation{
		ChatId:   chatId,
		ChatName: inviteName,
		Status:   userCallState,
	}
	invitation.Avatar = services.GetAvatar(userCallOwnerAvatar, tetATet)

	return c.JSON(http.StatusOK, invitation)
}
