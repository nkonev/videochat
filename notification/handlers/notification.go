package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/notification/auth"
	"nkonev.name/notification/db"
	. "nkonev.name/notification/logger"
	"nkonev.name/notification/utils"
)

type NotificationHandler struct {
	db db.DB
}

func NewMessageHandler(dbR db.DB) *NotificationHandler {
	return &NotificationHandler{
		db: dbR,
	}
}

func (mc *NotificationHandler) GetNotifications(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	if notifications, err := mc.db.GetNotifications(userPrincipalDto.UserId); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error get notification from db %v", err)
		return err
	} else {
		return c.JSON(http.StatusOK, notifications)
	}
}
