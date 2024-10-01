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

type NotificationsWrapper struct {
	Data  []dto.NotificationDto `json:"items"`
	Count int64                 `json:"count"` // total notification number for this user
}

type NotificationsCount struct {
	Count int64 `json:"totalCount"` // total notification number for this user
}

func (mc *NotificationHandler) GetNotifications(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	page := utils.FixPageString(c.QueryParam("page"))
	size := utils.FixSizeString(c.QueryParam("size"))
	offset := utils.GetOffset(page, size)

	if notifications, err := mc.db.GetNotifications(c.Request().Context(), userPrincipalDto.UserId, size, offset); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error get notification from db %v", err)
		return err
	} else {

		notificationsCount, err := mc.db.GetNotificationCount(c.Request().Context(), userPrincipalDto.UserId)
		if err != nil {
			return errors.New("Error during getting user chat count")
		}

		return c.JSON(http.StatusOK, NotificationsWrapper{Data: notifications, Count: notificationsCount})
	}
}

func (mc *NotificationHandler) GetNotificationsCount(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	notificationsCount, err := mc.db.GetNotificationCount(c.Request().Context(), userPrincipalDto.UserId)
	if err != nil {
		return errors.New("Error during getting user notification count")
	}

	return c.JSON(http.StatusOK, NotificationsCount{Count: notificationsCount})
}

func (mc *NotificationHandler) DeleteAllNotifications(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	err := mc.db.ClearAllNotifications(c.Request().Context(), userPrincipalDto.UserId)
	if err != nil {
		return errors.New("Error during getting user chat count")
	}

	err = mc.rabbitEventsPublisher.Publish(c.Request().Context(), userPrincipalDto.UserId, nil, services.NotificationClearAll)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Unable to send notification delete %v", err)
	}

	return c.NoContent(http.StatusOK)
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

	deletedNotificationType, err := mc.db.DeleteNotification(c.Request().Context(), notificationId, userPrincipalDto.UserId)
	if err != nil {
		return err
	}

	count, err := mc.db.GetNotificationCount(c.Request().Context(), userPrincipalDto.UserId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Unable to count notification %v", err)
		return err
	}

	err = mc.rabbitEventsPublisher.Publish(c.Request().Context(), userPrincipalDto.UserId, dto.NewWrapperNotificationDeleteDto(notificationId, count, deletedNotificationType), services.NotificationDelete)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Unable to send notification delete %v", err)
	}

	return c.NoContent(http.StatusAccepted)
}

func (mc *NotificationHandler) GetGlobalNotificationSettings(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	notSett, err := mc.db.GetNotificationGlobalSettings(c.Request().Context(), userPrincipalDto.UserId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting notification settings %v", err)
		return err
	}

	return c.JSON(http.StatusOK, notSett)
}

func (mc *NotificationHandler) PutGlobalNotificationSettings(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	var bindTo = new(dto.NotificationGlobalSettings)
	err := c.Bind(bindTo)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during reading notification settings %v", err)
		return err
	}

	err = mc.db.InitGlobalNotificationSettings(c.Request().Context(), userPrincipalDto.UserId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during initializing notification settings %v", err)
		return err
	}

	err = mc.db.PutNotificationGlobalSettings(c.Request().Context(), userPrincipalDto.UserId, bindTo)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during writing notification settings %v", err)
		return err
	}

	notSett, err := mc.db.GetNotificationGlobalSettings(c.Request().Context(), userPrincipalDto.UserId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting notification settings %v", err)
		return err
	}

	return c.JSON(http.StatusOK, notSett)
}

func (mc *NotificationHandler) GetChatNotificationSettings(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	notSett, err := mc.db.GetNotificationPerChatSettings(c.Request().Context(), userPrincipalDto.UserId, chatId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting notification settings %v", err)
		return err
	}

	return c.JSON(http.StatusOK, notSett)
}

func (mc *NotificationHandler) PutChatNotificationSettings(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	var bindTo = new(dto.NotificationPerChatSettings)
	err := c.Bind(bindTo)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during reading notification settings %v", err)
		return err
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	err = mc.db.InitPerChatNotificationSettings(c.Request().Context(), userPrincipalDto.UserId, chatId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during initializing notification settings %v", err)
		return err
	}

	err = mc.db.PutNotificationPerChatSettings(c.Request().Context(), userPrincipalDto.UserId, chatId, bindTo)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during writing notification settings %v", err)
		return err
	}

	notSett, err := mc.db.GetNotificationPerChatSettings(c.Request().Context(), userPrincipalDto.UserId, chatId)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting notification settings %v", err)
		return err
	}

	return c.JSON(http.StatusOK, notSett)
}
