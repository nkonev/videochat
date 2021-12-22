package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/notifications"
	"nkonev.name/chat/utils"
)

type VideoHandler struct {
	db          db.DB
	notificator notifications.Notifications
}

func NewVideoHandler(db db.DB, notificator notifications.Notifications) *VideoHandler {
	return &VideoHandler{db, notificator}
}

type simpleChat struct {
	Id        int64  `json:"id"`
	Name      string `json:"name"`
	IsTetATet bool   `json:"tetATet"`
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

func (vh *VideoHandler) NotifyAboutCallInvitation(c echo.Context) error {
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
		Id:        chat.Id,
		Name:      chat.Title,
		IsTetATet: chat.TetATet,
	}
	utils.ReplaceChatNameToLoginForTetATet(
		sch,
		&meAsUser,
		userId,
	)

	vh.notificator.NotifyAboutCallInvitation(c, chatId, userId, sch.GetName())
	return c.NoContent(http.StatusOK)
}
