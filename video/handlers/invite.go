package handlers

import (
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
}

const MissedCall = "missed_call"

func NewInviteHandler(dialService *services.DialRedisRepository, chatClient *client.RestClient, dialStatusPublisher *producer.RabbitDialStatusPublisher, notificationPublisher *producer.RabbitNotificationsPublisher, userService *services.UserService, chatDialerService *tasks.ChatDialerService) *InviteHandler {
	return &InviteHandler{
		dialRedisRepository:   dialService,
		chatClient:            chatClient,
		dialStatusPublisher:   dialStatusPublisher,
		notificationPublisher: notificationPublisher,
		userService:           userService,
		chatDialerService:     chatDialerService,
	}
}

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

	callee, err0 := utils.ParseInt64(c.QueryParam("userId"))
	if err0 != nil {
		return err0
	}

	call, err0 := utils.ParseBoolean(c.QueryParam("call"))
	if err0 != nil {
		return err0
	}

	// check my access to chat
	if ok, err := vh.chatClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	return c.NoContent(vh.addToCalling(c, callee, call, chatId, userPrincipalDto))
}

func (vh *InviteHandler) addToCalling(c echo.Context, callee int64, call bool, chatId int64, userPrincipalDto *auth.AuthResult) int {
	// check participant's access to chat
	if ok, err := vh.chatClient.CheckAccess(callee, chatId, c.Request().Context()); err != nil {
		return http.StatusInternalServerError
	} else if !ok {
		return http.StatusUnauthorized
	}

	behalfUserId, err := vh.dialRedisRepository.GetDialMetadata(c.Request().Context(), chatId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}
	if behalfUserId == services.NoUser {
		// ok
	} else if userPrincipalDto.UserId != behalfUserId {
		logger.GetLogEntry(c.Request().Context()).Infof("Call already started in this chat %v by %v", chatId, behalfUserId)
		return http.StatusAccepted
	}

	if call {
		err = vh.dialRedisRepository.AddToDialList(c.Request().Context(), callee, chatId, userPrincipalDto.UserId)
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			return http.StatusInternalServerError
		}
	} else {
		err = vh.dialRedisRepository.RemoveFromDialList(c.Request().Context(), callee, chatId)
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			return http.StatusInternalServerError
		}

		var videoIsInvitingDto = dto.VideoIsInvitingDto{
			ChatId:       chatId,
			UserIds:      []int64{callee},
			Status:       false,
			BehalfUserId: userPrincipalDto.UserId,
		}
		err = vh.dialStatusPublisher.Publish(&videoIsInvitingDto)
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			return http.StatusInternalServerError
		}

		if inviteNames, err := vh.chatClient.GetChatNameForInvite(chatId, behalfUserId, []int64{callee}, c.Request().Context()); err != nil {
			Logger.Error(err, "Failed during getting chat invite names")
		} else if len(inviteNames) == 1 {
			chatName := inviteNames[0]
			// here send missed call notification
			var missedCall = dto.NotificationEvent{
				EventType:              MissedCall,
				ChatId:                 chatId,
				UserId:                 callee,
				MissedCallNotification: &dto.MissedCallNotification{chatName.Name},
				ByUserId:               userPrincipalDto.UserId,
				ByLogin:                userPrincipalDto.UserLogin,
			}
			err = vh.notificationPublisher.Publish(missedCall)
			if err != nil {
				logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
				return http.StatusInternalServerError
			}
		}
	}

	return http.StatusOK
}

func (vh *InviteHandler) ProcessDialStart(c echo.Context) error {
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

	// in this block we start calling in case tet-a-tet
	usersOfChat := basicChatInfo.ParticipantIds
	if basicChatInfo.TetATet && len(usersOfChat) > 0 {
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		var oppositeUser *int64
		for _, userId := range usersOfChat {
			if userId != userPrincipalDto.UserId {
				var deUid = userId
				oppositeUser = &deUid
				break
			}
		}

		// uniq users by userId
		usersOfVideo, err := vh.userService.GetVideoParticipants(chatId, c.Request().Context())
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		var oppositeUserOfVideo *int64
		for _, ou := range usersOfVideo {
			if ou != userPrincipalDto.UserId {
				var deOu = ou
				oppositeUserOfVideo = &deOu
				break
			}
		}

		// and we(behalf user) doesn't have incoming call
		if oppositeUserOfVideo == nil && oppositeUser != nil {
			// we should call the counterpart (opposite user)
			vh.addToCalling(c, *oppositeUser, true, chatId, userPrincipalDto)
		}
	}

	// duplicate "take the phone" (pressing green tube) which cancels ringing logic for opposite user (or myself)
	vh.cancelCallingLogic(c, chatId, userPrincipalDto)

	return c.NoContent(http.StatusOK)
}

func (vh *InviteHandler) ProcessCancelInvitation(c echo.Context) error {
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

	return c.NoContent(vh.cancelCallingLogic(c, chatId, userPrincipalDto))
}

func (vh *InviteHandler) cancelCallingLogic(c echo.Context, chatId int64, userPrincipalDto *auth.AuthResult) int {
	behalfUserId, err := vh.dialRedisRepository.GetDialMetadata(c.Request().Context(), chatId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}
	if behalfUserId == services.NoUser {
		return http.StatusOK
	}

	err = vh.dialRedisRepository.RemoveFromDialList(c.Request().Context(), userPrincipalDto.UserId, chatId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}

	var videoIsInvitingDto = dto.VideoIsInvitingDto{
		ChatId:       chatId,
		UserIds:      []int64{userPrincipalDto.UserId},
		Status:       false,
		BehalfUserId: behalfUserId,
	}
	err = vh.dialStatusPublisher.Publish(&videoIsInvitingDto)

	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func (vh *InviteHandler) ProcessAsOwnerLeave(c echo.Context) error {
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

	behalfUserId, err := vh.dialRedisRepository.GetDialMetadata(c.Request().Context(), chatId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if behalfUserId == services.NoUser {
		return c.NoContent(http.StatusOK)
	}

	if behalfUserId != userPrincipalDto.UserId {
		return c.NoContent(http.StatusOK)
	}

	usersToDial, err := vh.dialRedisRepository.GetUsersToDial(c.Request().Context(), chatId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	err = vh.dialRedisRepository.RemoveDial(c.Request().Context(), chatId)

	var videoIsInvitingDto = dto.VideoIsInvitingDto{
		ChatId:       chatId,
		UserIds:      usersToDial,
		Status:       false,
		BehalfUserId: behalfUserId,
	}
	err = vh.dialStatusPublisher.Publish(&videoIsInvitingDto)

	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if chatNames, err := vh.chatClient.GetChatNameForInvite(chatId, behalfUserId, usersToDial, c.Request().Context()); err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
	} else {
		for _, chatName := range chatNames {
			// here send missed call notification
			var missedCall = dto.NotificationEvent{
				EventType:              MissedCall,
				ChatId:                 chatId,
				UserId:                 chatName.UserId,
				MissedCallNotification: &dto.MissedCallNotification{chatName.Name},
				ByUserId:               userPrincipalDto.UserId,
				ByLogin:                userPrincipalDto.UserLogin,
			}
			err = vh.notificationPublisher.Publish(missedCall)
			if err != nil {
				logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			}
		}
	}

	return c.NoContent(http.StatusOK)
}

func (vh *InviteHandler) AskDials(c echo.Context) error {
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

	vh.chatDialerService.SendDialStatusChanged(c.Request().Context(), userPrincipalDto.UserId, chatId)
	return c.NoContent(http.StatusOK)
}
