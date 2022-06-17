package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
)

type UserHandler struct {
	chatClient  *client.RestClient
	userService *services.UserService
}

func NewUserHandler(chatClient *client.RestClient, userService *services.UserService) *UserHandler {
	return &UserHandler{chatClient: chatClient, userService: userService}
}

type CountUsersResponse struct {
	UsersCount int64 `json:"usersCount"`
}

func (h *UserHandler) GetVideoUsers(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.chatClient.CheckAccess(userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	var roomName = utils.GetRoomNameFromId(chatId)
	usersCount, err := h.userService.CountUsers(c.Request().Context(), roomName)
	if err != nil {
		Logger.Errorf("got error during getting participants from http users request, %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, CountUsersResponse{UsersCount: usersCount})
}
