package handlers

import (
	"errors"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/notifications"
	"nkonev.name/chat/utils"
	"time"
)

type EditMessageDto struct {
	Id   int64  `json:"id"`
	Text string `json:"text"`
}

type CreateMessageDto struct {
	Text string `json:"text"`
}

type DisplayMessageDto = dto.DisplayMessageDto

func GetMessages(dbR db.DB, restClient client.RestClient) func(c echo.Context) error {
	return func(c echo.Context) error {
		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		page := utils.FixPageString(c.QueryParam("page"))
		size := utils.FixSizeString(c.QueryParam("size"))
		reverse := utils.GetBoolean(c.QueryParam("reverse"))
		offset := utils.GetOffset(page, size)

		chatIdString := c.Param("id")
		chatId, err := utils.ParseInt64(chatIdString)
		if err != nil {
			return err
		}

		if messages, err := dbR.GetMessages(chatId, userPrincipalDto.UserId, size, offset, reverse); err != nil {
			GetLogEntry(c.Request()).Errorf("Error get messages from db %v", err)
			return err
		} else {
			var ownersSet = map[int64]bool{}
			for _, c := range messages {
				ownersSet[c.OwnerId] = true
			}
			var owners = getUsersRemotelyOrEmpty(ownersSet, restClient, c)

			messageDtos := make([]*DisplayMessageDto, 0)
			for _, c := range messages {
				messageDtos = append(messageDtos, convertToMessageDto(c, owners))
			}

			GetLogEntry(c.Request()).Infof("Successfully returning %v messages", len(messageDtos))
			return c.JSON(200, messageDtos)
		}
	}
}

func GetMessage(dbR db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		chatId, err := GetPathParamAsInt64(c, "id")
		if err != nil {
			return err
		}

		messageId, err := GetPathParamAsInt64(c, "messageId")
		if err != nil {
			return err
		}

		if message, err := dbR.GetMessage(chatId, userPrincipalDto.UserId, messageId); err != nil {
			GetLogEntry(c.Request()).Errorf("Error get messages from db %v", err)
			return err
		} else {
			if message == nil {
				return c.NoContent(http.StatusNotFound)
			}
			messageDto := convertToMessageDto(message, map[int64]*dto.User{})
			GetLogEntry(c.Request()).Infof("Successfully returning message %v", messageDto)
			return c.JSON(200, messageDto)
		}
	}
}

func convertToMessageDto(dbMessage *db.Message, owners map[int64]*dto.User) *DisplayMessageDto {
	user := owners[dbMessage.OwnerId]
	return &DisplayMessageDto{
		Id:             dbMessage.Id,
		Text:           dbMessage.Text,
		ChatId:         dbMessage.ChatId,
		OwnerId:        dbMessage.OwnerId,
		CreateDateTime: dbMessage.CreateDateTime,
		EditDateTime:   dbMessage.EditDateTime,
		Owner:          user,
	}
}

func (a *CreateMessageDto) Validate() error {
	return validation.ValidateStruct(a, validation.Field(&a.Text, validation.Required, validation.Length(1, 1024*1024)))
}

func (a *EditMessageDto) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Text, validation.Required, validation.Length(1, 1024*1024)),
		validation.Field(&a.Id, validation.Required),
	)
}

func PostMessage(dbR db.DB, policy *bluemonday.Policy, notificator notifications.Notifications, restClient client.RestClient) func(c echo.Context) error {
	return func(c echo.Context) error {
		var bindTo = new(CreateMessageDto)
		if err := c.Bind(bindTo); err != nil {
			GetLogEntry(c.Request()).Errorf("Error during binding to dto %v", err)
			return err
		}

		if valid, err := ValidateAndRespondError(c, bindTo); err != nil || !valid {
			return err
		}

		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		chatId, err := GetPathParamAsInt64(c, "id")
		if err != nil {
			return err
		}

		errOuter := db.Transact(dbR, func(tx *db.Tx) error {
			if participant, err := tx.IsParticipant(userPrincipalDto.UserId, chatId); err != nil {
				return err
			} else if !participant {
				return c.JSON(http.StatusBadRequest, &utils.H{"message": "You are not allowed to write to this chat"})
			}
			id, createDatetime, editDatetime, err := tx.CreateMessage(convertToCreatableMessage(bindTo, userPrincipalDto, chatId, policy))
			if err != nil {
				return err
			}
			if tx.AddMessageRead(id, userPrincipalDto.UserId) != nil {
				return err
			}
			if tx.UpdateLastDatetimeChat(chatId) != nil {
				return err
			}

			var ownerIds []int64
			ownerIds = append(ownerIds, userPrincipalDto.UserId)
			users, err := restClient.GetUsers(ownerIds, c.Request().Context())
			if err != nil {
				GetLogEntry(c.Request()).Errorf("Unable to get user %v", err)
			}
			var maybeUser *dto.User = nil
			if len(users) != 0 {
				maybeUser = users[0]
			}

			dm := &DisplayMessageDto{
				Id:             id,
				Text:           bindTo.Text,
				ChatId:         chatId,
				OwnerId:        userPrincipalDto.UserId,
				CreateDateTime: createDatetime,
				EditDateTime:   editDatetime,
				Owner:          maybeUser,
			}

			notificator.NotifyAboutNewMessage(c, chatId, dm, userPrincipalDto)
			return c.JSON(http.StatusCreated, dm)
		})
		if errOuter != nil {
			GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
		}
		return errOuter
	}
}

func convertToCreatableMessage(dto *CreateMessageDto, authPrincipal *auth.AuthResult, chatId int64, policy *bluemonday.Policy) *db.Message {
	return &db.Message{
		Text:    policy.Sanitize(dto.Text),
		ChatId:  chatId,
		OwnerId: authPrincipal.UserId,
	}
}

func EditMessage(dbR db.DB, policy *bluemonday.Policy) func(c echo.Context) error {
	return func(c echo.Context) error {
		var bindTo = new(EditMessageDto)
		if err := c.Bind(bindTo); err != nil {
			GetLogEntry(c.Request()).Errorf("Error during binding to dto %v", err)
			return err
		}

		if valid, err := ValidateAndRespondError(c, bindTo); err != nil || !valid {
			return err
		}

		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		chatId, err := GetPathParamAsInt64(c, "id")
		if err != nil {
			return err
		}

		errOuter := db.Transact(dbR, func(tx *db.Tx) error {
			err := tx.EditMessage(convertToEditableMessage(bindTo, userPrincipalDto, chatId, policy))
			if err != nil {
				return err
			}
			return c.JSON(http.StatusCreated, &utils.H{"id": bindTo.Id})
		})
		if errOuter != nil {
			GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
		}
		return errOuter
	}
}

func convertToEditableMessage(dto *EditMessageDto, authPrincipal *auth.AuthResult, chatId int64, policy *bluemonday.Policy) *db.Message {
	return &db.Message{
		Id:           dto.Id,
		Text:         policy.Sanitize(dto.Text),
		ChatId:       chatId,
		OwnerId:      authPrincipal.UserId,
		EditDateTime: null.TimeFrom(time.Now()),
	}
}

func DeleteMessage(dbR db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		chatId, err := GetPathParamAsInt64(c, "id")
		if err != nil {
			return err
		}

		messageId, err := GetPathParamAsInt64(c, "messageId")
		if err != nil {
			return err
		}

		if err := dbR.DeleteMessage(messageId, userPrincipalDto.UserId, chatId); err != nil {
			return err
		} else {
			return c.JSON(http.StatusAccepted, &utils.H{"id": messageId})
		}
	}
}
