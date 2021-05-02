package handlers

import (
	"errors"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	strip "github.com/grokify/html-strip-tags-go"
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
	"strings"
	"time"
)

type EditMessageDto struct {
	Id   int64  `json:"id"`
	Text string `json:"text"`
}

type CreateMessageDto struct {
	Text string `json:"text"`
}

type MessageHandler struct {
	db          db.DB
	policy      *bluemonday.Policy
	notificator notifications.Notifications
	restClient  client.RestClient
}

func NewMessageHandler(dbR db.DB, policy *bluemonday.Policy, notificator notifications.Notifications, restClient client.RestClient) MessageHandler {
	return MessageHandler{
		db: dbR, policy: policy, notificator: notificator, restClient: restClient,
	}
}

func (mc MessageHandler) GetMessages(c echo.Context) error {
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

	if messages, err := mc.db.GetMessages(chatId, userPrincipalDto.UserId, size, offset, reverse); err != nil {
		GetLogEntry(c.Request()).Errorf("Error get messages from db %v", err)
		return err
	} else {
		var ownersSet = map[int64]bool{}
		for _, c := range messages {
			ownersSet[c.OwnerId] = true
		}
		var owners = getUsersRemotelyOrEmpty(ownersSet, mc.restClient, c)

		messageDtos := make([]*dto.DisplayMessageDto, 0)
		for _, c := range messages {
			messageDtos = append(messageDtos, convertToMessageDto(c, owners, userPrincipalDto))
		}

		GetLogEntry(c.Request()).Infof("Successfully returning %v messages", len(messageDtos))
		return c.JSON(200, messageDtos)
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

func (mc MessageHandler) GetMessage(c echo.Context) error {
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

	message, err := getMessage(c, &mc.db, mc.restClient, chatId, messageId, userPrincipalDto)
	if err != nil {
		return err
	}
	if message == nil {
		return c.NoContent(http.StatusNotFound)
	}
	GetLogEntry(c.Request()).Infof("Successfully returning message %v", message)
	return c.JSON(200, message)
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

func noContent(c echo.Context) error {
	return c.NoContent(204)
}

func (mc MessageHandler) PostMessage(c echo.Context) error {
	var bindTo = new(CreateMessageDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request()).Warnf("Error during binding to dto %v", err)
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

	errOuter := db.Transact(mc.db, func(tx *db.Tx) error {
		if participant, err := tx.IsParticipant(userPrincipalDto.UserId, chatId); err != nil {
			return err
		} else if !participant {
			return c.JSON(http.StatusBadRequest, &utils.H{"message": "You are not allowed to write to this chat"})
		}
		creatableMessage := convertToCreatableMessage(bindTo, userPrincipalDto, chatId, mc.policy)
		if creatableMessage.Text == "" {
			GetLogEntry(c.Request()).Infof("Empty message doesn't save")
			return noContent(c)
		}
		id, _, _, err := tx.CreateMessage(creatableMessage)
		if err != nil {
			return err
		}
		if tx.AddMessageRead(id, userPrincipalDto.UserId, chatId) != nil {
			return err
		}
		if tx.UpdateChatLastDatetimeChat(chatId) != nil {
			return err
		}

		participantIds, err := tx.GetParticipantIds(chatId)
		if err != nil {
			return err
		}
		message, err := getMessage(c, tx, mc.restClient, chatId, id, userPrincipalDto)
		if err != nil {
			return err
		}
		mc.notificator.NotifyAboutNewMessage(c, participantIds, chatId, message)
		mc.notificator.ChatNotifyMessageCount(participantIds, c, chatId, tx)
		return c.JSON(http.StatusCreated, message)
	})
	if errOuter != nil {
		GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func sanitizeMessage(policy *bluemonday.Policy, input string) string {
	return policy.Sanitize(input)
}

func convertToCreatableMessage(dto *CreateMessageDto, authPrincipal *auth.AuthResult, chatId int64, policy *bluemonday.Policy) *db.Message {
	return &db.Message{
		Text:    trim(sanitizeMessage(policy, dto.Text)),
		ChatId:  chatId,
		OwnerId: authPrincipal.UserId,
	}
}

func (mc MessageHandler) EditMessage(c echo.Context) error {
	var bindTo = new(EditMessageDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request()).Warnf("Error during binding to dto %v", err)
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

	errOuter := db.Transact(mc.db, func(tx *db.Tx) error {
		editableMessage := convertToEditableMessage(bindTo, userPrincipalDto, chatId, mc.policy)
		if editableMessage.Text == "" {
			GetLogEntry(c.Request()).Infof("Empty message doesn't save")
			return noContent(c)
		}
		err := tx.EditMessage(editableMessage)
		if err != nil {
			return err
		}
		ids, err := tx.GetParticipantIds(chatId)
		if err != nil {
			return err
		}

		message, err := getMessage(c, tx, mc.restClient, chatId, bindTo.Id, userPrincipalDto)
		if err != nil {
			return err
		}
		mc.notificator.NotifyAboutEditMessage(c, ids, chatId, message)

		return c.JSON(http.StatusCreated, &utils.H{"id": bindTo.Id})
	})
	if errOuter != nil {
		GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func convertToEditableMessage(dto *EditMessageDto, authPrincipal *auth.AuthResult, chatId int64, policy *bluemonday.Policy) *db.Message {
	return &db.Message{
		Id:           dto.Id,
		Text:         trim(sanitizeMessage(policy, dto.Text)),
		ChatId:       chatId,
		OwnerId:      authPrincipal.UserId,
		EditDateTime: null.TimeFrom(time.Now()),
	}
}

func trim(str string) string {
	return strings.TrimSpace(str)
}

func (mc MessageHandler) DeleteMessage(c echo.Context) error {
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

	if err := mc.db.DeleteMessage(messageId, userPrincipalDto.UserId, chatId); err != nil {
		return err
	} else {
		cd := &dto.DisplayMessageDto{
			Id: messageId,
		}
		if ids, err := mc.db.GetParticipantIds(chatId); err != nil {
			return err
		} else {
			mc.notificator.NotifyAboutDeleteMessage(c, ids, chatId, cd)
		}
		return c.JSON(http.StatusAccepted, &utils.H{"id": messageId})
	}
}

func (mc MessageHandler) TypeMessage(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	if participant, err := mc.db.IsParticipant(userPrincipalDto.UserId, chatId); err != nil {
		GetLogEntry(c.Request()).Errorf("Error during checking participant")
		return err
	} else if !participant {
		GetLogEntry(c.Request()).Infof("User %v is not participant of chat %v, skipping", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusAccepted)
	}

	var ownersSet = map[int64]bool{}
	ownersSet[userPrincipalDto.UserId] = true
	var owners = getUsersRemotelyOrEmpty(ownersSet, mc.restClient, c)
	typingUser := owners[userPrincipalDto.UserId]

	mc.notificator.NotifyAboutMessageTyping(c, chatId, typingUser)
	return c.NoContent(http.StatusAccepted)
}

type BroadcastDto struct {
	Text string `json:"text"`
}

func (mc MessageHandler) BroadcastMessage(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	if !userPrincipalDto.HasRole("ROLE_ADMIN") {
		return c.NoContent(http.StatusUnauthorized)
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	if participant, err := mc.db.IsParticipant(userPrincipalDto.UserId, chatId); err != nil {
		GetLogEntry(c.Request()).Errorf("Error during checking participant")
		return err
	} else if !participant {
		GetLogEntry(c.Request()).Infof("User %v is not participant of chat %v, skipping", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusAccepted)
	}

	var bindTo = new(BroadcastDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request()).Warnf("Error during binding to dto %v", err)
		return err
	}


	mc.notificator.NotifyAboutBroadcast(c, chatId, userPrincipalDto.UserId, userPrincipalDto.UserLogin, strip.StripTags(bindTo.Text))
	return c.NoContent(http.StatusAccepted)
}