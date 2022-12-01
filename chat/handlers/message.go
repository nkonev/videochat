package handlers

import (
	"context"
	"errors"
	"fmt"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"math"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
	"nkonev.name/chat/utils"
	"strings"
	"time"
)

type EditMessageDto struct {
	Id           int64      `json:"id"`
	Text         string     `json:"text"`
	FileItemUuid *uuid.UUID `json:"fileItemUuid"`
}

type CreateMessageDto struct {
	Text         string     `json:"text"`
	FileItemUuid *uuid.UUID `json:"fileItemUuid"`
}

type MessageHandler struct {
	db                 db.DB
	policy             *SanitizerPolicy
	stripSourceContent *StripSourcePolicy
	stripAllTags       *StripTagsPolicy
	notificator        services.Events
	restClient         client.RestClient
}

func NewMessageHandler(dbR db.DB, policy *SanitizerPolicy, stripSourceContent *StripSourcePolicy, stripAllTags *StripTagsPolicy, notificator services.Events, restClient client.RestClient) *MessageHandler {
	return &MessageHandler{
		db: dbR, policy: policy, stripSourceContent: stripSourceContent, stripAllTags: stripAllTags, notificator: notificator, restClient: restClient,
	}
}

func (mc *MessageHandler) GetMessages(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	var startingFromItemId int64
	startingFromItemIdString := c.QueryParam("startingFromItemId")
	if startingFromItemIdString == "" {
		startingFromItemId = math.MaxInt64
	} else {
		startingFromItemId2, err := utils.ParseInt64(startingFromItemIdString) // exclusive
		if err != nil {
			return err
		}
		startingFromItemId = startingFromItemId2
	}
	size := utils.FixSizeString(c.QueryParam("size"))
	reverse := utils.GetBoolean(c.QueryParam("reverse"))
	searchString := c.QueryParam("searchString")
	searchString = TrimAmdSanitize(mc.policy, searchString)
	hasHash := utils.GetBoolean(c.QueryParam("hasHash"))

	chatIdString := c.Param("id")
	chatId, err := utils.ParseInt64(chatIdString)
	if err != nil {
		return err
	}

	if messages, err := mc.db.GetMessages(chatId, userPrincipalDto.UserId, size, startingFromItemId, reverse, hasHash, searchString); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error get messages from db %v", err)
		return err
	} else {
		if hasHash {
			_, err = mc.db.AddMessageRead(startingFromItemId, userPrincipalDto.UserId, chatId)
			if err != nil {
				return err
			}
		}

		var ownersSet = map[int64]bool{}
		for _, c := range messages {
			ownersSet[c.OwnerId] = true
		}
		var owners = getUsersRemotelyOrEmpty(ownersSet, mc.restClient, c)
		messageDtos := make([]*dto.DisplayMessageDto, 0)
		for _, c := range messages {
			messageDtos = append(messageDtos, convertToMessageDto(c, owners, userPrincipalDto.UserId))
		}

		GetLogEntry(c.Request().Context()).Infof("Successfully returning %v messages", len(messageDtos))
		return c.JSON(http.StatusOK, messageDtos)
	}
}

func getMessage(c echo.Context, co db.CommonOperations, restClient client.RestClient, chatId int64, messageId int64, behalfUserId int64) (*dto.DisplayMessageDto, error) {
	if message, err := co.GetMessage(chatId, behalfUserId, messageId); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error get messages from db %v", err)
		return nil, err
	} else {
		if message == nil {
			return nil, nil
		}
		var ownersSet = map[int64]bool{}
		ownersSet[behalfUserId] = true
		var owners = getUsersRemotelyOrEmpty(ownersSet, restClient, c)
		return convertToMessageDto(message, owners, behalfUserId), nil
	}
}

func (mc *MessageHandler) GetMessage(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
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

	message, err := getMessage(c, &mc.db, mc.restClient, chatId, messageId, userPrincipalDto.UserId)
	if err != nil {
		return err
	}
	if message == nil {
		return c.NoContent(http.StatusNotFound)
	}
	GetLogEntry(c.Request().Context()).Infof("Successfully returning message %v", message)
	return c.JSON(http.StatusOK, message)
}

func convertToMessageDto(dbMessage *db.Message, owners map[int64]*dto.User, behalfUserId int64) *dto.DisplayMessageDto {
	user := owners[dbMessage.OwnerId]
	if user == nil {
		user = &dto.User{Login: fmt.Sprintf("user%v", dbMessage.OwnerId), Id: dbMessage.OwnerId}
	}
	ret := &dto.DisplayMessageDto{
		Id:             dbMessage.Id,
		Text:           dbMessage.Text,
		ChatId:         dbMessage.ChatId,
		OwnerId:        dbMessage.OwnerId,
		CreateDateTime: dbMessage.CreateDateTime,
		EditDateTime:   dbMessage.EditDateTime,
		Owner:          user,
		FileItemUuid:   dbMessage.FileItemUuid,
	}

	ret.SetPersonalizedFields(behalfUserId)

	return ret
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

func (mc *MessageHandler) PostMessage(c echo.Context) error {
	var bindTo = new(CreateMessageDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	if valid, err := ValidateAndRespondError(c, bindTo); err != nil || !valid {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
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
			GetLogEntry(c.Request().Context()).Infof("Empty message doesn't save")
			return noContent(c)
		}
		id, _, _, err := tx.CreateMessage(creatableMessage)
		if err != nil {
			return err
		}
		_, err = tx.AddMessageRead(id, userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if tx.UpdateChatLastDatetimeChat(chatId) != nil {
			return err
		}

		participantIds, err := tx.GetAllParticipantIds(chatId)
		if err != nil {
			return err
		}
		message, err := getMessage(c, tx, mc.restClient, chatId, id, userPrincipalDto.UserId)
		if err != nil {
			return err
		}

		var users = getUsersRemotelyOrEmptyFromSlice(participantIds, mc.restClient, c)
		var addedMentions, strippedText = mc.findMentions(message.Text, users, c.Request().Context())
		var reallyAddedMentions = excludeMyself(addedMentions, userPrincipalDto)
		mc.notificator.NotifyAddMention(c, reallyAddedMentions, chatId, message.Id, strippedText)

		mc.notificator.NotifyAboutNewMessage(c, participantIds, chatId, message)
		mc.notificator.ChatNotifyMessageCount(participantIds, c, chatId, tx)
		return c.JSON(http.StatusCreated, message)
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func convertToCreatableMessage(dto *CreateMessageDto, authPrincipal *auth.AuthResult, chatId int64, policy *SanitizerPolicy) *db.Message {
	return &db.Message{
		Text:         TrimAmdSanitize(policy, dto.Text),
		ChatId:       chatId,
		OwnerId:      authPrincipal.UserId,
		FileItemUuid: dto.FileItemUuid,
	}
}

func (mc *MessageHandler) EditMessage(c echo.Context) error {
	var bindTo = new(EditMessageDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	if valid, err := ValidateAndRespondError(c, bindTo); err != nil || !valid {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	errOuter := db.Transact(mc.db, func(tx *db.Tx) error {
		editableMessage := convertToEditableMessage(bindTo, userPrincipalDto, chatId, mc.policy)
		if editableMessage.Text == "" {
			GetLogEntry(c.Request().Context()).Infof("Empty message doesn't save")
			return noContent(c)
		}

		participantIds, err := tx.GetAllParticipantIds(chatId)
		if err != nil {
			return err
		}

		var users = getUsersRemotelyOrEmptyFromSlice(participantIds, mc.restClient, c)

		oldMessage, err := tx.GetMessage(chatId, userPrincipalDto.UserId, editableMessage.Id)
		if err != nil {
			return err
		}
		var oldMentions, _ = mc.findMentions(oldMessage.Text, users, c.Request().Context())

		err = tx.EditMessage(editableMessage)
		if err != nil {
			return err
		}

		message, err := getMessage(c, tx, mc.restClient, chatId, bindTo.Id, userPrincipalDto.UserId)
		if err != nil {
			return err
		}

		var newMentions, strippedText = mc.findMentions(message.Text, users, c.Request().Context())

		var userIdsToNotifyAboutMentionCreated []int64
		var userIdsToNotifyAboutMentionDeleted []int64

		for _, oldMentionedUserId := range oldMentions {
			if !utils.Contains(newMentions, oldMentionedUserId) {
				userIdsToNotifyAboutMentionDeleted = append(userIdsToNotifyAboutMentionDeleted, oldMentionedUserId)
			}
		}

		for _, newMentionUserId := range newMentions {
			if !utils.Contains(oldMentions, newMentionUserId) {
				userIdsToNotifyAboutMentionCreated = append(userIdsToNotifyAboutMentionCreated, newMentionUserId)
			}
		}

		var reallyAddedMentions = excludeMyself(userIdsToNotifyAboutMentionCreated, userPrincipalDto)
		mc.notificator.NotifyAddMention(c, reallyAddedMentions, chatId, message.Id, strippedText)
		mc.notificator.NotifyRemoveMention(c, userIdsToNotifyAboutMentionDeleted, chatId, message.Id)

		mc.notificator.NotifyAboutEditMessage(c, participantIds, chatId, message)

		return c.JSON(http.StatusCreated, &utils.H{"id": bindTo.Id})
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func excludeMyself(mentionedUserIds []int64, principalDto *auth.AuthResult) []int64 {
	var result = []int64{}
	for _, userId := range mentionedUserIds {
		if principalDto != nil && userId != principalDto.UserId {
			result = append(result, userId)
		}
	}
	return result
}

func convertToEditableMessage(dto *EditMessageDto, authPrincipal *auth.AuthResult, chatId int64, policy *SanitizerPolicy) *db.Message {
	return &db.Message{
		Id:           dto.Id,
		Text:         TrimAmdSanitize(policy, dto.Text),
		ChatId:       chatId,
		OwnerId:      authPrincipal.UserId,
		EditDateTime: null.TimeFrom(time.Now()),
		FileItemUuid: dto.FileItemUuid,
	}
}

func (mc *MessageHandler) DeleteMessage(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
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
	oldMessage, err := mc.db.GetMessage(chatId, userPrincipalDto.UserId, messageId)
	if err != nil {
		return err
	}
	participantIds, err := mc.db.GetAllParticipantIds(chatId)
	if err != nil {
		return err
	}
	var users = getUsersRemotelyOrEmptyFromSlice(participantIds, mc.restClient, c)

	if err := mc.db.DeleteMessage(messageId, userPrincipalDto.UserId, chatId); err != nil {
		return err
	} else {
		var oldMentions, _ = mc.findMentions(oldMessage.Text, users, c.Request().Context())
		mc.notificator.NotifyRemoveMention(c, oldMentions, chatId, messageId)

		cd := &dto.DisplayMessageDto{
			Id:     messageId,
			ChatId: chatId,
		}
		mc.notificator.NotifyAboutDeleteMessage(c, participantIds, chatId, cd)

		return c.JSON(http.StatusAccepted, &utils.H{"id": messageId})
	}
}

func (mc MessageHandler) ReadMessage(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
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

	_, err = mc.db.AddMessageRead(messageId, userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	mc.notificator.NotifyRemoveMention(c, []int64{userPrincipalDto.UserId}, chatId, messageId)

	return c.NoContent(http.StatusAccepted)
}

func (mc *MessageHandler) TypeMessage(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	if participant, err := mc.db.IsParticipant(userPrincipalDto.UserId, chatId); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking participant")
		return err
	} else if !participant {
		GetLogEntry(c.Request().Context()).Infof("User %v is not participant of chat %v, skipping", userPrincipalDto.UserId, chatId)
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

func (mc *MessageHandler) BroadcastMessage(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	if participant, err := mc.db.IsAdmin(userPrincipalDto.UserId, chatId); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking participant")
		return err
	} else if !participant {
		GetLogEntry(c.Request().Context()).Infof("User %v is not participant of chat %v, skipping", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusAccepted)
	}

	var bindTo = new(BroadcastDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	mc.notificator.NotifyAboutMessageBroadcast(c, chatId, userPrincipalDto.UserId, userPrincipalDto.UserLogin, mc.stripAllTags.Sanitize(bindTo.Text))
	return c.NoContent(http.StatusAccepted)
}

func (mc *MessageHandler) RemoveFileItem(c echo.Context) error {
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}
	userId, err := GetQueryParamAsInt64(c, "userId")
	if err != nil {
		return err
	}
	isParticipant, err := mc.db.IsParticipant(userId, chatId)
	if err != nil {
		return err
	}
	if !isParticipant {
		msg := "user " + c.QueryParam("userId") + " is not belongs to chat " + c.QueryParam("chatId")
		GetLogEntry(c.Request().Context()).Warnf(msg)
		return c.JSON(http.StatusAccepted, &utils.H{"message": msg})
	}
	fileItemUuid := c.QueryParam("fileItemUuid")
	messageId, err := mc.db.SetFileItemUuidToNull(userId, chatId, fileItemUuid)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Unable to set FileItemUuid to full for fileItemUuid=%v, chatId=%v", fileItemUuid, chatId)
		return c.NoContent(http.StatusInternalServerError)
	}

	// notifying
	ids, err := mc.db.GetAllParticipantIds(chatId)
	if err != nil {
		return err
	}
	message, err := getMessage(c, &mc.db, mc.restClient, chatId, messageId, userId)
	if err != nil {
		return err
	}
	mc.notificator.NotifyAboutEditMessage(c, ids, chatId, message)

	return c.NoContent(http.StatusOK)
}

type Tuple struct {
	MinioKey string `json:"minioKey"`
	Filename string `json:"filename"`
	Exists   bool   `json:"exists"`
}

func (rec Tuple) String() string {
	return fmt.Sprintf("Tuple(key=%s, exists=%v, filename=%s)", rec.MinioKey, rec.Exists, rec.Filename)
}

func (mc *MessageHandler) CheckEmbeddedFiles(c echo.Context) error {
	requestMap := new(map[int64][]*Tuple)
	if err := c.Bind(requestMap); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}
	GetLogEntry(c.Request().Context()).Debugf("Got request %v", requestMap)
	for chatIdKey, tupleValues := range *requestMap {
		GetLogEntry(c.Request().Context()).Infof("Processing %v=%v", chatIdKey, tupleValues)
		exists, err := mc.db.IsChatExists(chatIdKey)
		if err != nil {
			Logger.Warnf("Error during checking existence of %v, skipping: %v", chatIdKey, err)
			continue
		}
		if !exists {
			for _, value := range tupleValues {
				value.Exists = false
			}
			Logger.Infof("Set not exists for all files for chatId = %v", chatIdKey)
			continue
		}

		// find here all files in messages
		var filenames = []string{}
		for _, tupleValue := range tupleValues {
			filenames = append(filenames, tupleValue.Filename)
		}
		messageIdsAndTextsFromDb, err := mc.db.IsEmbedExists(chatIdKey, filenames)
		if err != nil {
			Logger.Warnf("Error during checking existence of filenames %v in %v, skipping: %v", filenames, chatIdKey, err)
			continue
		}

		// invert to false for all pairs
		for _, value := range tupleValues {
			value.Exists = false
		}
		// here we try to find it in message texts
		for _, messageIdsAndTextPair := range messageIdsAndTextsFromDb {
			for _, tupleValue := range tupleValues {
				if strings.Contains(messageIdsAndTextPair.Text, tupleValue.Filename) {
					Logger.Infof("File %v is exists in chat with id %v", tupleValue.Filename, chatIdKey)
					tupleValue.Exists = true
				}
			}
		}
	}

	return c.JSON(http.StatusOK, requestMap)
}

func (mc *MessageHandler) findMentions(messageText string, users map[int64]*dto.User, c context.Context) ([]int64, string) {
	var result = []int64{}
	withoutSourceTags := mc.stripSourceContent.Sanitize(messageText)
	for _, user := range users {
		if strings.Contains(withoutSourceTags, "@"+user.Login) {
			result = append(result, user.Id)
		}
	}
	withoutAnyHtml := mc.stripAllTags.Sanitize(withoutSourceTags)
	return result, withoutAnyHtml
}
