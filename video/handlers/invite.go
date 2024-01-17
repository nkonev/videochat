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
	"nkonev.name/video/tasks"
	"nkonev.name/video/utils"
)

type InviteHandler struct {
	dialRedisRepository   *services.DialRedisRepository
	chatClient            *client.RestClient
	dialStatusPublisher   *producer.RabbitDialStatusPublisher
	notificationPublisher *producer.RabbitNotificationsPublisher
	userService           *services.UserService
	chatDialerService     *tasks.ChatDialerService
	chatInvitationService *services.ChatInvitationService
	stateChangedEventService *services.StateChangedEventService
}

const EventMissedCall = "missed_call"

func NewInviteHandler(dialService *services.DialRedisRepository, chatClient *client.RestClient, dialStatusPublisher *producer.RabbitDialStatusPublisher, notificationPublisher *producer.RabbitNotificationsPublisher, userService *services.UserService, chatDialerService *tasks.ChatDialerService, chatInvitationService *services.ChatInvitationService, stateChangedEventService *services.StateChangedEventService) *InviteHandler {
	return &InviteHandler{
		dialRedisRepository:   dialService,
		chatClient:            chatClient,
		dialStatusPublisher:   dialStatusPublisher,
		notificationPublisher: notificationPublisher,
		userService:           userService,
		chatDialerService:     chatDialerService,
		chatInvitationService: chatInvitationService,
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

	addToCallCall, err := utils.ParseBoolean(c.QueryParam("call"))
	if err != nil {
		return err
	}

	// check my access to chat
	if ok, err := vh.chatClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	if (addToCallCall) {
		return c.NoContent(vh.addToCalling(c, callee, chatId, userPrincipalDto))
	} else {
		return c.NoContent(vh.removeFromCalling(c, callee, chatId, userPrincipalDto))
	}
}

func (vh *InviteHandler) checkAccessOverCall(ctx context.Context, callee int64, chatId int64, userPrincipalDto *auth.AuthResult) (bool, int) {
	// check participant's access to chat
	if ok, err := vh.chatClient.CheckAccess(callee, chatId, ctx); err != nil {
		return false, http.StatusInternalServerError
	} else if !ok {
		return false, http.StatusUnauthorized
	}

	ownerId, err := vh.dialRedisRepository.GetDialMetadata(ctx, chatId)
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error %v", err)
		return false, http.StatusInternalServerError
	}
	if ownerId == services.NoUser {
		// ok
	} else if userPrincipalDto.UserId != ownerId {
		logger.GetLogEntry(ctx).Infof("Call already started in this chat %v by %v", chatId, ownerId)
		return false, http.StatusForbidden
	}
	return true, http.StatusOK
}

func (vh *InviteHandler) addToCalling(c echo.Context, callee int64, chatId int64, userPrincipalDto *auth.AuthResult) int {
	ok, code := vh.checkAccessOverCall(c.Request().Context(), callee, chatId, userPrincipalDto)
	if !ok {
		return code
	}

	status, err := vh.dialRedisRepository.GetUserCallStatus(c.Request().Context(), callee)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}

	if !services.CanOverrideCallStatus(status) {
		return http.StatusConflict
	}

	// we remove callee's previous inviting - only after CanOverrideCallStatus() check
	vh.removePrevious(c, callee)

	err = vh.dialRedisRepository.AddToDialList(c.Request().Context(), callee, chatId, userPrincipalDto.UserId, services.CallStatusInviting)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}

	// for better user experience
	vh.sendEvents(c, chatId, []int64{callee}, services.CallStatusInviting, userPrincipalDto.UserId)

	return http.StatusOK
}

func (vh *InviteHandler) removeFromCalling(c echo.Context, callee int64, chatId int64, userPrincipalDto *auth.AuthResult) int {
	ok, code := vh.checkAccessOverCall(c.Request().Context(), callee, chatId, userPrincipalDto)
	if !ok {
		return code
	}

	code = vh.removeFromCallingList(c, chatId, []int64{callee}, services.CallStatusRemoving)
	if code != http.StatusOK {
		return code
	}

	// if we remove user from call - send them EventMissedCall notification
	vh.sendMissedCallNotification(chatId, c.Request().Context(), userPrincipalDto, []int64{callee})

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

	// first of all we remove our previous inviting
	vh.removePrevious(c, userPrincipalDto.UserId)

	maybeOwnerId, err := vh.dialRedisRepository.GetDialMetadata(c.Request().Context(), chatId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error during getting OwnerId %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	// if videochat does not exist
	// then it means that I'm the owner and I need to create it
	if maybeOwnerId == services.NoUser {
		// and put myself with a status "inCall"
		err = vh.dialRedisRepository.AddToDialList(c.Request().Context(), userPrincipalDto.UserId, chatId, userPrincipalDto.UserId, services.CallStatusInCall)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during adding as owner %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		// during entering into dial. Returns status: true which means that frontend should (initially) draw the calling.
		// Now it used only in tet-a-tet.
		// If we are in the tet-a-tet
		basicChatInfo, err := vh.chatClient.GetBasicChatInfo(chatId, userPrincipalDto.UserId, c.Request().Context()) // tet-a-tet
		if err != nil {
			return err
		}

		usersOfChat := basicChatInfo.ParticipantIds

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
				vh.addToCalling(c, *oppositeUser, chatId, userPrincipalDto)
			}
		}
	} else { // we enter to somebody's chat
		err = vh.dialRedisRepository.AddToDialList(c.Request().Context(), userPrincipalDto.UserId, chatId, maybeOwnerId, services.CallStatusInCall)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during adding as non-owner %v", err)
			return err
		}
		vh.sendEvents(c, chatId, []int64{userPrincipalDto.UserId}, services.CallStatusInCall, maybeOwnerId)
	}

	return c.NoContent(http.StatusOK)
}

func (vh *InviteHandler) removePrevious(c echo.Context, userId int64) {
	previousUserCallState, previousChatId, _, _, err := vh.dialRedisRepository.GetUserCallState(c.Request().Context(), userId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Unable to get user call state %v", err)
	}
	if previousUserCallState != services.CallStatusNotFound {
		vh.dialRedisRepository.RemoveFromDialList(c.Request().Context(), userId, previousChatId)
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

	return c.NoContent(vh.removeFromCallingList(c, chatId, []int64{userPrincipalDto.UserId}, services.CallStatusCancelling))
}

// question: how not to overwhelm the system by iterating over all the users and all the chats ?
// answer: using opened rooms and rooms are going to be closed - see livekit's room.empty_timeout


func (vh *InviteHandler) removeFromCallingList(c echo.Context, chatId int64, usersOfDial []int64, callStatus string) int {
	ownerId, err := vh.dialRedisRepository.GetDialMetadata(c.Request().Context(), chatId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}
	if ownerId == services.NoUser {
		return http.StatusOK
	}

	// we remove callee by setting status
	for _, userId := range usersOfDial {
		err = vh.setUserStatus(c.Request().Context(), userId, chatId, callStatus)
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		}
	}
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}

	vh.sendEvents(c, chatId, usersOfDial, callStatus, ownerId)

	return http.StatusOK
}

func (vh *InviteHandler) sendEvents(c echo.Context, chatId int64, usersOfDial []int64, callStatus string, ownerId int64) {
	// we send "stop-inviting-for-userPrincipalDto.UserId-signal" to the call's owner
	vh.dialStatusPublisher.Publish(chatId, getMap(usersOfDial, callStatus), ownerId)

	// send the new status immediately to user
	vh.chatInvitationService.SendInvitationsWithStatuses(c.Request().Context(), chatId, ownerId, getMap(usersOfDial, callStatus))
}

func getMap(userIds []int64, status string) map[int64]string {
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

	ownerId, err := vh.dialRedisRepository.GetDialMetadata(c.Request().Context(), chatId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if ownerId == services.NoUser {
		return c.NoContent(http.StatusOK)
	}

	if ownerId == userPrincipalDto.UserId { // owner leaving
		usersToDial, err := vh.dialRedisRepository.GetUsersOfDial(c.Request().Context(), chatId)
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		videoParticipants, err := vh.userService.GetVideoParticipants(chatId, c.Request().Context())
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		missedUsers := make([]int64, 0)
		inVideoUsers := make([]int64, 0)
		for _, redisCallUser := range usersToDial {
			// sometimes in race-condition manner due to livekit the call owner can be here, so we remove them by not to adding
			if redisCallUser != ownerId {
				if !utils.Contains(videoParticipants, redisCallUser) {
					missedUsers = append(missedUsers, redisCallUser)
				} else {
					inVideoUsers = append(inVideoUsers, redisCallUser)
				}
			}
		}

		// the owner removes all the dials by setting status
		toRemove := make([]int64, 0)
		for _, u := range missedUsers {
			toRemove = append(toRemove, u)
		}
		toRemove = append(toRemove, userPrincipalDto.UserId)
		vh.removeFromCallingList(c, chatId, toRemove, services.CallStatusRemoving)

		// for all participants to dial - send EventMissedCall notification
		vh.sendMissedCallNotification(chatId, c.Request().Context(), userPrincipalDto, missedUsers)

		// delegate ownership to another user
		vh.dialRedisRepository.TransferOwnership(c.Request().Context(), inVideoUsers, userPrincipalDto.UserId, chatId)
	} else {
		vh.removeFromCallingList(c, chatId, []int64{userPrincipalDto.UserId}, services.CallStatusRemoving)
	}

	return c.NoContent(http.StatusOK)
}

func (vh *InviteHandler) sendMissedCallNotification(chatId int64, ctx context.Context, userPrincipalDto *auth.AuthResult, usersToDial []int64) {
	if len(usersToDial) > 0 {
		if chatNames, err := vh.chatClient.GetChatNameForInvite(chatId, userPrincipalDto.UserId, usersToDial, ctx); err != nil {
			logger.GetLogEntry(ctx).Errorf("Error %v", err)
		} else {
			for _, chatName := range chatNames {
				// here send missed call notification
				var missedCall = dto.NotificationEvent{
					EventType:              EventMissedCall,
					ChatId:                 chatId,
					UserId:                 chatName.UserId,
					MissedCallNotification: &dto.MissedCallNotification{chatName.Name},
					ByUserId:               userPrincipalDto.UserId,
					ByLogin:                userPrincipalDto.UserLogin,
				}
				err = vh.notificationPublisher.Publish(missedCall)
				if err != nil {
					logger.GetLogEntry(ctx).Errorf("Error %v", err)
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

	userIdsToDial, err := vh.dialRedisRepository.GetUsersOfDial(c.Request().Context(), chatId)
	if err != nil {
		Logger.Warnf("Error %v", err)
		return c.NoContent(http.StatusOK)
	}

	var statuses = vh.chatDialerService.GetStatuses(c.Request().Context(), chatId, userIdsToDial)

	err = vh.dialStatusPublisher.Publish(chatId, statuses, userPrincipalDto.UserId)
	if err != nil {
		Logger.Error(err, "Failed during marshal VideoIsInvitingDto")
		return c.NoContent(http.StatusOK)
	}

	vh.stateChangedEventService.NotifyAllChatsAboutUsersVideoStatus(c.Request().Context())

	return c.NoContent(http.StatusOK)
}
