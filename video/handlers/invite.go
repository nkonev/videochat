package handlers

import (
	"encoding/json"
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/utils"
)

type InviteHandler struct {
	rabbitMqPublisher *producer.RabbitInvitePublisher
	chatClient        *client.RestClient
}

func NewInviteHandler(rabbitMqPublisher *producer.RabbitInvitePublisher, chatClient *client.RestClient) *InviteHandler {
	return &InviteHandler{
		rabbitMqPublisher: rabbitMqPublisher,
		chatClient:        chatClient,
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

	inviteDto := dto.VideoInviteDto{
		ChatId:       chatId,
		UserId:       userId,
		BehalfUserId: userPrincipalDto.UserId,
		BehalfLogin:  userPrincipalDto.UserLogin,
	}

	marshal, err := json.Marshal(inviteDto)
	if err != nil {
		Logger.Error(err, "Failed during marshal chatNotifyDto")
		return err
	}

	err = vh.rabbitMqPublisher.Publish(marshal)
	if err != nil {
		Logger.Error(err, "Failed during marshal chatNotifyDto")
		return err
	}

	return c.NoContent(http.StatusOK)
}
