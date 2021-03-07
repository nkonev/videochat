package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/notifications"
	"nkonev.name/chat/utils"
	"time"
)

type VideoHandler struct {
	db          db.DB
	restClient  client.RestClient
	notificator notifications.Notifications
}

func NewVideoHandler(db db.DB, restClient client.RestClient, notificator notifications.Notifications) VideoHandler {
	return VideoHandler{db, restClient, notificator}
}

func (vh VideoHandler) NotifyAboutVideoCallChange(c echo.Context) error {
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}

	usersCount, err := GetQueryParamAsInt64(c, "usersCount")
	if err != nil {
		return err
	}

	vh.notificator.NotifyAboutVideoCallChanged(c, chatId, usersCount)
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

func (vh VideoHandler) NotifyAboutKick(c echo.Context) error {
	chatId, err := utils.ParseInt64(c.QueryParam("chatId"))
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
		return c.JSON(http.StatusAccepted, &utils.H{"message": "user " + c.QueryParam("userId") + " is not belongs to chat " + c.QueryParam("chatId")})
	}

	vh.notificator.NotifyAboutKick(c, chatId, userId)
	return c.NoContent(200)

}

func (vh VideoHandler) Kick(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		logger.Logger.Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

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
		return c.JSON(http.StatusAccepted, &utils.H{"message": "user " + c.QueryParam("userId") + " is not belongs to chat " + c.QueryParam("chatId")})
	}

	admin, err := vh.db.IsAdmin(userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !admin {
		return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
	}

	vh.notificator.NotifyAboutKick(c, chatId, userId)

	go vh.kickVideoStreamWithWait(chatId, userId)

	return c.NoContent(200)
}

func (vh VideoHandler) kickVideoStreamWithWait(chatId, userId int64) {
	duration := viper.GetDuration("video.kickVideoAfter")
	if duration <= 0 {
		logger.Logger.Warnf("video.kickVideoAfter is not set, skipping invoking kickVideoStream()")
		return
	}
	time.Sleep(duration)
	vh.kickVideoStream(chatId, userId)
}

func (vh VideoHandler) kickVideoStream(chatId, userId int64) {
	logger.Logger.Infof("video kick chatId=%v, userId=%v", chatId, userId)
	err := vh.restClient.Kick(chatId, userId)
	if err != nil {
		logger.Logger.Warnf("Non-successful invoking video kick %v", err)
	}
}