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

			messageDtos := make([]*dto.DisplayMessageDto, 0)
			for _, c := range messages {
				messageDtos = append(messageDtos, convertToMessageDto(c, owners, userPrincipalDto))
			}

			GetLogEntry(c.Request()).Infof("Successfully returning %v messages", len(messageDtos))
			return c.JSON(200, messageDtos)
		}
	}
}

func getMessage(c echo.Context, co db.CommonOperations, restClient client.RestClient, chatId int64, messageId int64, userPrincipalDto *auth.AuthResult) (*dto.DisplayMessageDto, error) {
	if message, err := co.GetMessage(chatId, userPrincipalDto.UserId, messageId); err != nil {
		GetLogEntry(c.Request()).Errorf("Error get messages from db %v", err)
		return nil, err
	} else {
		if message == nil {
			return nil, nil
		}
		var ownersSet = map[int64]bool{}
		ownersSet[userPrincipalDto.UserId] = true
		var owners = getUsersRemotelyOrEmpty(ownersSet, restClient, c)
		return convertToMessageDto(message, owners, userPrincipalDto), nil
	}
}

func GetMessage(dbR db.DB, restClient client.RestClient) func(c echo.Context) error {
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

		message, err := getMessage(c, &dbR, restClient, chatId, messageId, userPrincipalDto)
		if err != nil {
			return err
		}
		if message == nil {
			return c.NoContent(http.StatusNotFound)
		}
		GetLogEntry(c.Request()).Infof("Successfully returning message %v", message)
		return c.JSON(200, message)
	}
}

func convertToMessageDto(dbMessage *db.Message, owners map[int64]*dto.User, userPrincipalDto *auth.AuthResult) *dto.DisplayMessageDto {
	user := owners[dbMessage.OwnerId]
	return &dto.DisplayMessageDto{
		Id:             dbMessage.Id,
		Text:           dbMessage.Text,
		ChatId:         dbMessage.ChatId,
		OwnerId:        dbMessage.OwnerId,
		CreateDateTime: dbMessage.CreateDateTime,
		EditDateTime:   dbMessage.EditDateTime,
		Owner:          user,
		CanEdit:        dbMessage.OwnerId == userPrincipalDto.UserId,
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
			id, _, _, err := tx.CreateMessage(convertToCreatableMessage(bindTo, userPrincipalDto, chatId, policy))
			if err != nil {
				return err
			}
			if tx.AddMessageRead(id, userPrincipalDto.UserId) != nil {
				return err
			}
			if tx.UpdateChatLastDatetimeChat(chatId) != nil {
				return err
			}

			participantIds, err := tx.GetParticipantIds(chatId)
			if err != nil {
				return err
			}
			message, err := getMessage(c, tx, restClient, chatId, id, userPrincipalDto)
			if err != nil {
				return err
			}
			notificator.NotifyAboutNewMessage(c, participantIds, chatId, message)
			return c.JSON(http.StatusCreated, message)
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

func EditMessage(dbR db.DB, policy *bluemonday.Policy, notificator notifications.Notifications, restClient client.RestClient) func(c echo.Context) error {
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
			ids, err := tx.GetParticipantIds(chatId)
			if err != nil {
				return err
			}

			message, err := getMessage(c, tx, restClient, chatId, bindTo.Id, userPrincipalDto)
			if err != nil {
				return err
			}
			notificator.NotifyAboutEditMessage(c, ids, chatId, message)

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

func DeleteMessage(dbR db.DB, notificator notifications.Notifications) func(c echo.Context) error {
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
			cd := &dto.DisplayMessageDto{
				Id: messageId,
			}
			if ids, err := dbR.GetParticipantIds(chatId); err != nil {
				return err
			} else {
				notificator.NotifyAboutDeleteMessage(c, ids, chatId, cd)
			}
			return c.JSON(http.StatusAccepted, &utils.H{"id": messageId})
		}
	}
}
