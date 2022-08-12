package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	"nkonev.name/video/logger"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
)

type InviteHandler struct {
	dialService *services.DialRedisService
	chatClient  *client.RestClient
}

func NewInviteHandler(dialService *services.DialRedisService, chatClient *client.RestClient) *InviteHandler {
	return &InviteHandler{
		dialService: dialService,
		chatClient:  chatClient,
	}
}

func (vh *InviteHandler) NotifyAboutCallInvitation(c echo.Context) error {
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

	if call {
		err = vh.dialService.AddToDialList(c.Request().Context(), userId, chatId, userPrincipalDto.UserId, userPrincipalDto.UserLogin)
	} else {
		err = vh.dialService.RemoveFromDialList(c.Request().Context(), userId, chatId)
	}
	if err != nil {
		logger.GetLogEntry(c.Request().Context()).Errorf("Error %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
