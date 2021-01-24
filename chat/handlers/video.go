package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/notifications"
	"nkonev.name/chat/utils"
)

type VideoHandler struct {
	db         db.DB
	restClient client.RestClient
	notificator notifications.Notifications
}

func NewVideoHandler(db db.DB, restClient client.RestClient, notificator notifications.Notifications) VideoHandler {
	return VideoHandler{db, restClient, notificator}
}

func (vh VideoHandler) GetOpenviduConfig(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}
	openviduWsUrl := getLoadBalancedOpenvidu(chatId)
	return c.JSON(http.StatusOK, &utils.H{
		"wsUrl": openviduWsUrl,
	})
}

func getLoadBalancedOpenvidu(chatId int64) string {
	return viper.GetString("openvidu.wsUrl")
}

func (vh VideoHandler) GetOpenviduToken(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	isParticipant, err := vh.db.IsParticipant(userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !isParticipant {
		return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no acces to this chat"})
	}

	session, err := vh.restClient.CreateOpenviduSession(chatId)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Unable to create openvidu session for chat %v %v", chatId, err)
		return err
	}
	token, err := vh.restClient.CreateOpenviduConnection(session)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Unable to create openvidu connection for chat %v %v", chatId, err)
		return err
	}
	return c.JSON(http.StatusOK, &utils.H{
		"token": token,
	})
}

func getUsersCount(vh VideoHandler, chatId int64, c echo.Context) (int32, error) {
	info, err := vh.restClient.GetOpenviduSessionInfo(chatId)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Unable to get session info for chat %v %v", chatId, err)
		return 0, err
	}
	var count int32 = 0
	if info != nil {
		count = info.Connections.NumberOfElements
	}
	return count, nil
}

func (vh VideoHandler) GetUsersCount(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}
	var _, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	count, err := getUsersCount(vh, chatId, c)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Unable to get session info for chat %v %v", chatId, err)
		return c.JSON(http.StatusOK, &utils.H{"usersCount": 0})
	}
	return c.JSON(http.StatusOK, &utils.H{"usersCount": count})
}

func (vh VideoHandler) NotifyAboutVideoCallChange(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	count, err := getUsersCount(vh, chatId, c)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Unable to get session info for chat %v %v", chatId, err)
		return err
	}

	vh.notificator.NotifyAboutVideoCallChanged(c, chatId, count)
	return c.NoContent(200)
}

func (vh VideoHandler) NotifyAboutCallInvitation(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	userId, err := utils.ParseInt64(c.QueryParam("userId"))
	if err != nil {
		return err
	}

	isParticipant, err := vh.db.IsParticipant(userId, chatId)
	if err != nil {
		return err
	}
	if !isParticipant {
		return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no acces to this chat"})
	}

	vh.notificator.NotifyAboutCallInvitation(c, chatId, userId)
	return c.NoContent(200)
}