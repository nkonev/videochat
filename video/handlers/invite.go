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
	"nkonev.name/video/utils"
)

type InviteHandler struct {
	dialRedisRepository *services.DialRedisRepository
	chatClient          *client.RestClient
	dialStatusPublisher *producer.RabbitDialStatusPublisher
}

func NewInviteHandler(dialService *services.DialRedisRepository, chatClient *client.RestClient, dialStatusPublisher *producer.RabbitDialStatusPublisher) *InviteHandler {
	return &InviteHandler{
		dialRedisRepository: dialService,
		chatClient:          chatClient,
		dialStatusPublisher: dialStatusPublisher,
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

	userId, err0 := utils.ParseInt64(c.QueryParam("userId"))
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

	// check participant's access to chat
	if ok, err := vh.chatClient.CheckAccess(userId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	behalfUserId, _, err := vh.dialRedisRepository.GerDialMetadata(c.Request().Context(), chatId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if behalfUserId == services.NoUser {
		// ok
	} else if userPrincipalDto.UserId != behalfUserId {
		logger.GetLogEntry(c.Request().Context()).Infof("Call already started in this chat %v by %v", chatId, behalfUserId)
		return c.NoContent(http.StatusAccepted)
	}

	if call {
		err = vh.dialRedisRepository.AddToDialList(c.Request().Context(), userId, chatId, userPrincipalDto.UserId, userPrincipalDto.UserLogin)
	} else {
		err = vh.dialRedisRepository.RemoveFromDialList(c.Request().Context(), userId, chatId)
		if err != nil {
			logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		var videoIsInvitingDto = dto.VideoIsInvitingDto{
			ChatId:       chatId,
			UserIds:      []int64{userId},
			Status:       false,
			BehalfUserId: userPrincipalDto.UserId,
		}
		err = vh.dialStatusPublisher.Publish(&videoIsInvitingDto)
	}
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

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

	behalfUserId, _, err := vh.dialRedisRepository.GerDialMetadata(c.Request().Context(), chatId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	err = vh.dialRedisRepository.RemoveFromDialList(c.Request().Context(), userPrincipalDto.UserId, chatId)
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return c.NoContent(http.StatusInternalServerError)
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
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
