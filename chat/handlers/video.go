package handlers

import (
	"encoding/json"
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/notifications"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/utils"
	"time"
)

type VideoHandler struct {
	db          db.DB
	notificator notifications.Notifications
	producer *producer.RabbitPublisher
}

func NewVideoHandler(db db.DB, notificator notifications.Notifications, producer *producer.RabbitPublisher) VideoHandler {
	return VideoHandler{db, notificator, producer}
}

type simpleChat struct {
	Id                 int64     `json:"id"`
	Name               string    `json:"name"`
	IsTetATet			   bool 	 `json:"tetATet"`
}

func (r *simpleChat) GetId() int64 {
	return r.Id
}

func (r *simpleChat) GetName() string {
	return r.Name
}

func (r *simpleChat) SetName(s string) {
	r.Name = s
}

func (r *simpleChat) GetIsTetATet() bool {
	return r.IsTetATet
}

func (vh VideoHandler) NotifyAboutCallInvitation(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		logger.Logger.Errorf("Error during getting auth context")
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
	if isParticipant, err := vh.db.IsParticipant(userPrincipalDto.UserId, chatId); err != nil {
		return err
	} else if !isParticipant {
		return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
	}

	// check participant's access to chat
	if isParticipant, err := vh.db.IsParticipant(userId, chatId); err != nil {
		return err
	} else if !isParticipant {
		return c.JSON(http.StatusUnauthorized, &utils.H{"message": "User have no access to this chat"})
	}

	chat, err := vh.db.GetChat(userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}

	meAsUser := dto.User{Id: userPrincipalDto.UserId, Login: userPrincipalDto.UserLogin}
	var sch dto.ChatDtoWithTetATet = &simpleChat{
		Id: chat.Id,
		Name: chat.Title,
		IsTetATet: chat.TetATet,
	}
	ReplaceChatNameToLoginForTetATet(
		sch,
		&meAsUser,
		userId,
	)

	vh.notificator.NotifyAboutCallInvitation(c, chatId, userId, sch.GetName())
	return c.NoContent(http.StatusOK)
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

	admin, err := vh.db.IsAdmin(userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !admin {
		return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
	}

	vh.notificator.NotifyAboutKick(c, chatId, userId)

	go vh.kickVideoStreamWithWait(chatId, userId)

	return c.NoContent(http.StatusOK)
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

type KickUserDto struct {
	ChatId int64 `json:"chatId"`
	UserId int64 `json:"userId"`
}

// It's control shot for video microservice.
// It will kick user forcibly if user's frontend didn't received message from centrifuge.
func (vh VideoHandler) kickVideoStream(chatId, userId int64) {
	logger.Logger.Infof("video kick chatId=%v, userId=%v", chatId, userId)

	dto := KickUserDto{ChatId: chatId, UserId: userId}
	marshal, err := json.Marshal(dto)
	if err != nil {
		logger.Logger.Warnf("Non-successful marshalling video kick %v", err)
		return
	}

	if err = vh.producer.Publish(marshal); err != nil {
		logger.Logger.Warnf("Non-successful invoking video kick %v", err)
	}
}

func (vh VideoHandler) ForceMute(c echo.Context) error {
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

	admin, err := vh.db.IsAdmin(userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !admin {
		return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
	}

	vh.notificator.NotifyAboutForceMute(c, chatId, userId)

	return c.NoContent(http.StatusOK)
}
