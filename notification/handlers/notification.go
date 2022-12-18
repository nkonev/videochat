package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/notification/auth"
	"nkonev.name/notification/db"
	"nkonev.name/notification/dto"
	. "nkonev.name/notification/logger"
	"nkonev.name/notification/producer"
	"nkonev.name/notification/services"
	"nkonev.name/notification/utils"
)

type NotificationHandler struct {
	db                    *db.DB
	rabbitEventsPublisher *producer.RabbitEventPublisher
}

func NewMessageHandler(dbR *db.DB, rabbitEventsPublisher *producer.RabbitEventPublisher) *NotificationHandler {
	return &NotificationHandler{
		db:                    dbR,
		rabbitEventsPublisher: rabbitEventsPublisher,
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

func (mc *NotificationHandler) ReadNotification(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	notificationId, err := GetPathParamAsInt64(c, "notificationId")
	if err != nil {
		return err
	}

	err = mc.db.DeleteNotification(notificationId, userPrincipalDto.UserId)
	if err != nil {
		return err
	}

	err = mc.rabbitEventsPublisher.Publish(userPrincipalDto.UserId, dto.NewNotificationDeleteDto(notificationId), services.NotificationDelete, c.Request().Context())
	if err != nil {
		Logger.Errorf("Unable to send notification delete %v", err)
	}

	return c.NoContent(http.StatusAccepted)
}

func (mc *NotificationHandler) GetNotificationSettings(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	err := mc.db.InitNotificationSettings(userPrincipalDto.UserId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during initializing notification settings %v", err)
		return err
	}

	notSett, err := mc.db.GetNotificationSettings(userPrincipalDto.UserId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting notification settings %v", err)
		return err
	}

	return c.JSON(http.StatusOK, notSett)
}

func (mc *NotificationHandler) PutNotificationSettings(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	var bindTo = new(dto.NotificationSettings)
	err := c.Bind(bindTo)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during reading notification settings %v", err)
		return err
	}

	err = mc.db.InitNotificationSettings(userPrincipalDto.UserId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during initializing notification settings %v", err)
		return err
	}

	err = mc.db.PutNotificationSettings(userPrincipalDto.UserId, bindTo)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during writing notification settings %v", err)
		return err
	}

	notSett, err := mc.db.GetNotificationSettings(userPrincipalDto.UserId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting notification settings %v", err)
		return err
	}

	return c.JSON(http.StatusOK, notSett)
}
