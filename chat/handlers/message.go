package handlers

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/getlantern/deepcopy"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"math"
	"net/http"
	"net/url"
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

const maxMessageLen = 1024 * 1024
const minMessageLen = 1
const allUsers = "all"
const hereUsers = "here"

const NonExistentUser = -65000
const DeletedUser = -1
const AllUsers = -2
const HereUsers = -3
const badMediaUrl = "BAD_MEDIA_URL"

type EditMessageDto struct {
	Id int64 `json:"id"`
	CreateMessageDto
}

type CreateMessageDto struct {
	Text                string                   `json:"text"`
	BlogPost            bool                     `json:"blogPost"`
	FileItemUuid        *uuid.UUID               `json:"fileItemUuid"`
	EmbedMessageRequest *dto.EmbedMessageRequest `json:"embedMessage"`
}

type MessageHandler struct {
	db                 *db.DB
	policy             *services.SanitizerPolicy
	stripSourceContent *services.StripSourcePolicy
	stripAllTags       *services.StripTagsPolicy
	notificator        services.Events
	restClient         *client.RestClient
}

func NewMessageHandler(dbR *db.DB, policy *services.SanitizerPolicy, stripSourceContent *services.StripSourcePolicy, stripAllTags *services.StripTagsPolicy, notificator services.Events, restClient *client.RestClient) *MessageHandler {
	return &MessageHandler{
		db: dbR, policy: policy, stripSourceContent: stripSourceContent, stripAllTags: stripAllTags, notificator: notificator, restClient: restClient,
	}
}

func (mc *MessageHandler) FindMessageByFileItemUuid(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatIdString := c.Param("id")
	chatId, err := utils.ParseInt64(chatIdString)
	if err != nil {
		return err
	}


	isParticipant, err := mc.db.IsParticipant(userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !isParticipant {
		return c.NoContent(http.StatusUnauthorized)
	}

	fileItemUuid := c.Param("fileItemUuid")
	messageId, err := mc.db.FindMessageByFileItemUuid(chatId, fileItemUuid)
	if err != nil {
		return err
	}

	if messageId != db.MessageNotFoundId {
		return c.JSON(http.StatusOK, &utils.H{"messageId": messageId})
	} else {
		return c.NoContent(http.StatusNoContent)
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

	return db.Transact(mc.db, func(tx *db.Tx) error {
		isParticipant, err := tx.IsParticipant(userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !isParticipant {
			return c.NoContent(http.StatusUnauthorized)
		}

		if messages, err := tx.GetMessages(chatId, size, startingFromItemId, reverse, hasHash, searchString); err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error get messages from db %v", err)
			return err
		} else {
			if hasHash {
				err := mc.addMessageReadAndSendIt(tx, c, chatId, startingFromItemId, userPrincipalDto.UserId)
				if err != nil {
					return err
				}
			}

			var ownersSet = map[int64]bool{}
			var chatsPreSet = map[int64]bool{}
			for _, message := range messages {
				populateSets(message, ownersSet, chatsPreSet)
			}
			chatsSet, err := tx.GetChatsBasic(chatsPreSet, userPrincipalDto.UserId)
			if err != nil {
				return err
			}
			var owners = getUsersRemotelyOrEmpty(ownersSet, mc.restClient, c)
			messageDtos := make([]*dto.DisplayMessageDto, 0)
			for _, c := range messages {
				messageDtos = append(messageDtos, convertToMessageDto(c, owners, chatsSet, userPrincipalDto.UserId))
			}

			GetLogEntry(c.Request().Context()).Infof("Successfully returning %v messages", len(messageDtos))
			return c.JSON(http.StatusOK, messageDtos)
		}
	})
}

func getMessage(c echo.Context, co db.CommonOperations, restClient *client.RestClient, chatId int64, messageId int64, behalfUserId int64) (*dto.DisplayMessageDto, error) {
	if message, err := co.GetMessage(chatId, behalfUserId, messageId); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error get messages from db %v", err)
		return nil, err
	} else {
		if message == nil {
			return nil, nil
		}
		var ownersSet = map[int64]bool{}
		var chatsPreSet = map[int64]bool{}
		populateSets(message, ownersSet, chatsPreSet)

		chatsSet, err := co.GetChatsBasic(chatsPreSet, behalfUserId)
		if err != nil {
			return nil, err
		}

		var owners = getUsersRemotelyOrEmpty(ownersSet, restClient, c)
		return convertToMessageDto(message, owners, chatsSet, behalfUserId), nil
	}
}

func populateSets(message *db.Message, ownersSet map[int64]bool, chatsPreSet map[int64]bool) {
	ownersSet[message.OwnerId] = true
	if message.ResponseEmbeddedMessageReplyOwnerId != nil {
		var embeddedMessageReplyOwnerId = *message.ResponseEmbeddedMessageReplyOwnerId
		ownersSet[embeddedMessageReplyOwnerId] = true
	} else if message.ResponseEmbeddedMessageResendOwnerId != nil {
		var embeddedMessageResendOwnerId = *message.ResponseEmbeddedMessageResendOwnerId
		ownersSet[embeddedMessageResendOwnerId] = true
		var embeddedMessageResendChatId = *message.ResponseEmbeddedMessageResendChatId
		chatsPreSet[embeddedMessageResendChatId] = true
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

	message, err := getMessage(c, mc.db, mc.restClient, chatId, messageId, userPrincipalDto.UserId)
	if err != nil {
		return err
	}
	if message == nil {
		return c.NoContent(http.StatusNotFound)
	}
	GetLogEntry(c.Request().Context()).Infof("Successfully returning message %v", message)
	return c.JSON(http.StatusOK, message)
}

func convertToMessageDto(dbMessage *db.Message, owners map[int64]*dto.User, chats map[int64]*db.BasicChatDtoExtended, behalfUserId int64) *dto.DisplayMessageDto {
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
		Pinned:         dbMessage.Pinned,
		BlogPost:       dbMessage.BlogPost,
	}
	ret.Text = patchStorageUrlToPreventCachingVideo(ret.Text)

	if dbMessage.ResponseEmbeddedMessageReplyOwnerId != nil {
		embeddedUser := owners[*dbMessage.ResponseEmbeddedMessageReplyOwnerId]
		ret.EmbedMessage = &dto.EmbedMessageResponse{
			Id:        *dbMessage.ResponseEmbeddedMessageReplyId,
			Text:      *dbMessage.ResponseEmbeddedMessageReplyText,
			EmbedType: *dbMessage.ResponseEmbeddedMessageType,
			Owner:     embeddedUser,
		}
	} else if dbMessage.ResponseEmbeddedMessageResendOwnerId != nil {
		embeddedUser := owners[*dbMessage.ResponseEmbeddedMessageResendOwnerId]
		basicChat := chats[*dbMessage.ResponseEmbeddedMessageResendChatId]
		var embedChatName *string = nil
		if !basicChat.IsTetATet {
			embedChatName = &basicChat.Title
		}

		ret.EmbedMessage = &dto.EmbedMessageResponse{
			Id:            *dbMessage.ResponseEmbeddedMessageResendId,
			ChatId:        dbMessage.ResponseEmbeddedMessageResendChatId,
			ChatName:      embedChatName,
			Text:          dbMessage.Text,
			EmbedType:     *dbMessage.ResponseEmbeddedMessageType,
			Owner:         embeddedUser,
			IsParticipant: basicChat.BehalfUserIsParticipant,
		}
		ret.Text = ""
	}

	ret.SetPersonalizedFields(behalfUserId)

	return ret
}

func (a *CreateMessageDto) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Text, validation.Required, validation.Length(minMessageLen, maxMessageLen)),
	)
}

func (a *EditMessageDto) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Text, validation.Required, validation.Length(minMessageLen, maxMessageLen)),
		validation.Field(&a.Id, validation.Required),
	)
}

func (mc *MessageHandler) PostMessage(c echo.Context) error {
	var bindTo = new(CreateMessageDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	if bindTo.EmbedMessageRequest == nil || (bindTo.EmbedMessageRequest != nil && bindTo.EmbedMessageRequest.EmbedType == dto.EmbedMessageTypeReply) {
		if valid, err := ValidateAndRespondError(c, bindTo); err != nil || !valid {
			return err
		}
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
		creatableMessage, err := convertToCreatableMessage(bindTo, userPrincipalDto, chatId, mc.policy)
		if err != nil {
			var mediaError *MediaUrlErr
			if errors.As(err, &mediaError) {
				return c.JSON(http.StatusBadRequest, &utils.H{"message": err.Error(), "businessErrorCode": badMediaUrl})
			} else {
				return c.JSON(http.StatusBadRequest, mediaError.Error())
			}
		}

		err = mc.validateAndSetEmbedFieldsEmbedMessage(tx, bindTo, creatableMessage)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during checking embed %v", err)
			return err
		}

		hasMessages, err := tx.HasMessages(chatId)
		if err != nil {
			return err
		}
		chatBasic, err := tx.GetChatBasic(chatId)
		if err != nil {
			return err
		}

		if !hasMessages && chatBasic.IsBlog {
			creatableMessage.BlogPost = true
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

		responseDto, err := getChat(tx, mc.restClient, c, chatId, userPrincipalDto.UserId, 0, 0)
		if err != nil {
			return err
		}
		copiedChat, err := getChatWithAdminedUsers(c, responseDto, tx)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}

		mc.notificator.NotifyAboutChangeChat(c, copiedChat, participantIds, tx)

		message, err := getMessage(c, tx, mc.restClient, chatId, id, userPrincipalDto.UserId)
		if err != nil {
			return err
		}

		chatNameForNotification, err := mc.getChatNameForNotification(tx, chatId)
		if err != nil {
			return err
		}

		var users = getUsersRemotelyOrEmptyFromSlice(participantIds, mc.restClient, c)
		var userOnlines = getUserOnlinesRemotelyOrEmptyFromSlice(participantIds, mc.restClient, c)
		var addedMentions, strippedText = mc.findMentions(message.Text, true, users, userOnlines)
		var reply, userToSendTo = mc.wasReplyAdded(nil, message, chatId)
		var reallyAddedMentions = excludeMyself(addedMentions, userPrincipalDto)
		mc.notificator.NotifyAddMention(c, reallyAddedMentions, chatId, message.Id, strippedText, userPrincipalDto.UserId, userPrincipalDto.UserLogin, chatNameForNotification)
		mc.notificator.NotifyAddReply(c, reply, userToSendTo, userPrincipalDto.UserId, userPrincipalDto.UserLogin, chatNameForNotification)
		mc.notificator.NotifyAboutNewMessage(c, participantIds, chatId, message)
		//mc.notificator.ChatNotifyMessageCount(participantIds, c, chatId, tx) - it's included in NotifyAboutChangeChat
		return c.JSON(http.StatusCreated, message)
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (mc *MessageHandler) getChatNameForNotification(tx *db.Tx, chatId int64) (string, error) {
	chatBasic, err := tx.GetChatBasic(chatId)
	if err != nil {
		return "", err
	}
	chatName := chatBasic.Title
	if chatBasic.IsTetATet {
		chatName = ""
	}
	return chatName, nil

}

func (mc *MessageHandler) validateAndSetEmbedFieldsEmbedMessage(tx *db.Tx, input *CreateMessageDto, receiver *db.Message) error {
	if input.EmbedMessageRequest != nil {
		if input.EmbedMessageRequest.Id == 0 {
			return errors.New("Missed embed message id")
		}
		if input.EmbedMessageRequest.EmbedType == "" {
			return errors.New("Missed embedMessageType")
		} else {
			if input.EmbedMessageRequest.EmbedType != dto.EmbedMessageTypeReply && input.EmbedMessageRequest.EmbedType != dto.EmbedMessageTypeResend {
				return errors.New("Wrong embedMessageType")
			}
			if input.EmbedMessageRequest.EmbedType == dto.EmbedMessageTypeResend && input.EmbedMessageRequest.ChatId == 0 {
				return errors.New("Missed embedChatId for EmbedMessageTypeResend")
			}
		}

		if input.EmbedMessageRequest.EmbedType == dto.EmbedMessageTypeReply {
			receiver.RequestEmbeddedMessageId = &input.EmbedMessageRequest.Id
			receiver.RequestEmbeddedMessageType = &input.EmbedMessageRequest.EmbedType
			return nil
		} else if input.EmbedMessageRequest.EmbedType == dto.EmbedMessageTypeResend {
			receiver.RequestEmbeddedMessageId = &input.EmbedMessageRequest.Id
			receiver.RequestEmbeddedMessageType = &input.EmbedMessageRequest.EmbedType
			// check if this input.EmbedChatId resendable
			chat, err := tx.GetChatBasic(input.EmbedMessageRequest.ChatId)
			if err != nil {
				return err
			}
			if !chat.CanResend {
				return errors.New("Resending is forbidden for this chat")
			}
			messageText, messageOwnerId, err := tx.GetMessageBasic(input.EmbedMessageRequest.ChatId, input.EmbedMessageRequest.Id)
			if err != nil {
				return err
			}
			if messageText == nil {
				return errors.New("Missing the message")
			}
			receiver.Text = *messageText
			receiver.RequestEmbeddedMessageOwnerId = messageOwnerId
			receiver.RequestEmbeddedMessageChatId = &input.EmbedMessageRequest.ChatId
			return nil
		}
		return errors.New("Unexpected branch, logical mistake")
	}

	return nil
}

func convertToCreatableMessage(dto *CreateMessageDto, authPrincipal *auth.AuthResult, chatId int64, policy *services.SanitizerPolicy) (*db.Message, error) {
	trimmedAndSanitized, err := TrimAmdSanitizeMessage(policy, dto.Text)
	if err != nil {
		return nil, err
	}
	return &db.Message{
		Text:         trimmedAndSanitized,
		ChatId:       chatId,
		OwnerId:      authPrincipal.UserId,
		FileItemUuid: dto.FileItemUuid,
	}, nil
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
		editableMessage, err := convertToEditableMessage(bindTo, userPrincipalDto, chatId, mc.policy)
		if err != nil {
			var mediaError *MediaUrlErr
			if errors.As(err, &mediaError) {
				return c.JSON(http.StatusBadRequest, &utils.H{"message": err.Error(), "businessErrorCode": badMediaUrl})
			} else {
				return c.JSON(http.StatusBadRequest, mediaError.Error())
			}
		}

		err = mc.validateAndSetEmbedFieldsEmbedMessage(tx, &bindTo.CreateMessageDto, editableMessage)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during checking embed %v", err)
			return err
		}

		participantIds, err := tx.GetAllParticipantIds(chatId)
		if err != nil {
			return err
		}

		var users = getUsersRemotelyOrEmptyFromSlice(participantIds, mc.restClient, c)
		var userOnlines = getUserOnlinesRemotelyOrEmptyFromSlice(participantIds, mc.restClient, c)

		oldMessage, err := tx.GetMessage(chatId, userPrincipalDto.UserId, editableMessage.Id)
		if err != nil {
			return err
		}
		var oldMentions, _ = mc.findMentions(oldMessage.Text, false, users, userOnlines)

		err = tx.EditMessage(editableMessage)
		if err != nil {
			return err
		}

		message, err := getMessage(c, tx, mc.restClient, chatId, bindTo.Id, userPrincipalDto.UserId)
		if err != nil {
			return err
		}

		var newMentions, strippedText = mc.findMentions(message.Text, true, users, userOnlines)

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

		chatNameForNotification, err := mc.getChatNameForNotification(tx, chatId)
		if err != nil {
			return err
		}

		var replyAdded, userToSendToAdded = mc.wasReplyAdded(oldMessage, message, chatId)
		mc.notificator.NotifyAddReply(c, replyAdded, userToSendToAdded, userPrincipalDto.UserId, userPrincipalDto.UserLogin, chatNameForNotification)
		var replyRemoved, userToSendRemoved = mc.wasReplyRemoved(oldMessage, message, chatId)
		mc.notificator.NotifyRemoveReply(c, replyRemoved, userToSendRemoved)

		var reallyAddedMentions = excludeMyself(userIdsToNotifyAboutMentionCreated, userPrincipalDto)
		mc.notificator.NotifyAddMention(c, reallyAddedMentions, chatId, message.Id, strippedText, userPrincipalDto.UserId, userPrincipalDto.UserLogin, chatNameForNotification)
		mc.notificator.NotifyRemoveMention(c, userIdsToNotifyAboutMentionDeleted, chatId, message.Id)

		mc.notificator.NotifyAboutEditMessage(c, participantIds, chatId, message)

		return c.JSON(http.StatusCreated, &utils.H{"id": bindTo.Id})
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

type SetFileItemUuid struct {
	FileItemUuid *string `json:"fileItemUuid"`
	MessageId int64 	`json:"messageId"`
}

func (mc *MessageHandler) SetFileItemUuid(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	isParticipant, err := mc.db.IsParticipant(userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !isParticipant {
		msg := "user " + utils.Int64ToString(userPrincipalDto.UserId) + " is not belongs to chat " + utils.Int64ToString(chatId)
		GetLogEntry(c.Request().Context()).Warnf(msg)
		return c.JSON(http.StatusAccepted, &utils.H{"message": msg})
	}

	bindTo := new(SetFileItemUuid)
	err = c.Bind(bindTo)
	if err != nil {
		return err
	}

	_, ownerId, err := mc.db.GetMessageBasic(chatId, bindTo.MessageId)
	if err != nil {
		return err
	}
	if ownerId == nil || *ownerId != userPrincipalDto.UserId {
		msg := "user " + utils.Int64ToString(userPrincipalDto.UserId) + " is not owner of message " + utils.Int64ToString(bindTo.MessageId)
		GetLogEntry(c.Request().Context()).Warnf(msg)
		return c.JSON(http.StatusAccepted, &utils.H{"message": msg})
	}

	err = mc.db.SetFileItemUuidTo(userPrincipalDto.UserId, chatId, bindTo.MessageId, bindTo.FileItemUuid)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Unable to set FileItemUuid to full for fileItemUuid=%v, chatId=%v", bindTo.FileItemUuid, chatId)
		return c.NoContent(http.StatusInternalServerError)
	}

	// notifying
	ids, err := mc.db.GetAllParticipantIds(chatId)
	if err != nil {
		return err
	}
	message, err := getMessage(c, mc.db, mc.restClient, chatId, bindTo.MessageId, userPrincipalDto.UserId)
	if err != nil {
		return err
	}
	mc.notificator.NotifyAboutEditMessage(c, ids, chatId, message)

	return c.NoContent(http.StatusOK)
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

func convertToEditableMessage(dto *EditMessageDto, authPrincipal *auth.AuthResult, chatId int64, policy *services.SanitizerPolicy) (*db.Message, error) {
	trimmedAndSanitized, err := TrimAmdSanitizeMessage(policy, dto.Text)
	if err != nil {
		return nil, err
	}
	return &db.Message{
		Id:           dto.Id,
		Text:         trimmedAndSanitized,
		ChatId:       chatId,
		OwnerId:      authPrincipal.UserId,
		EditDateTime: null.TimeFrom(time.Now()),
		FileItemUuid: dto.FileItemUuid,
		BlogPost:     dto.BlogPost,
	}, nil
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
	var userOnlines = getUserOnlinesRemotelyOrEmptyFromSlice(participantIds, mc.restClient, c)

	if err := mc.db.DeleteMessage(messageId, userPrincipalDto.UserId, chatId); err != nil {
		return err
	} else {
		var oldMentions, _ = mc.findMentions(oldMessage.Text, false, users, userOnlines)
		mc.notificator.NotifyRemoveMention(c, oldMentions, chatId, messageId)

		cd := &dto.DisplayMessageDto{
			Id:     messageId,
			ChatId: chatId,
		}
		mc.notificator.NotifyAboutDeleteMessage(c, participantIds, chatId, cd)

		var replyRemoved, userToSendRemoved = mc.wasReplyRemoved(oldMessage, nil, chatId)
		mc.notificator.NotifyRemoveReply(c, replyRemoved, userToSendRemoved)

		return c.JSON(http.StatusAccepted, &utils.H{"id": messageId})
	}
}

func (mc *MessageHandler) ReadMessage(c echo.Context) error {
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

	return db.Transact(mc.db, func(tx *db.Tx) error {
		// here we don't check message ownership because user can read foreign messages
		// (any user has their own last read message per chat)

		err := mc.addMessageReadAndSendIt(tx, c, chatId, messageId, userPrincipalDto.UserId)
		if err != nil {
			return err
		}
		mc.notificator.NotifyRemoveMention(c, []int64{userPrincipalDto.UserId}, chatId, messageId)
		mc.notificator.NotifyRemoveReply(c, &dto.ReplyDto{
			MessageId: messageId,
			ChatId:    chatId,
		}, &userPrincipalDto.UserId)
		return c.NoContent(http.StatusAccepted)
	})
}

func (mc *MessageHandler) addMessageReadAndSendIt(tx *db.Tx, c echo.Context, chatId int64, messageId int64, userId int64) error {
	_, err := tx.AddMessageRead(messageId, userId, chatId)
	if err != nil {
		return err
	}
	mc.notificator.ChatNotifyMessageCount([]int64{userId}, c, chatId, tx)

	return nil
}

func (mc *MessageHandler) GetReadMessageUsers(c echo.Context) error {
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

	page := utils.FixPageString(c.QueryParam("page"))
	size := utils.FixSizeString(c.QueryParam("size"))
	offset := utils.GetOffset(page, size)

	Logger.Debugf("Processing GetReadMessageUsers user %v, chatId %v, messageId %v", userPrincipalDto.UserId, chatId, messageId)

	if participant, err := mc.db.IsParticipant(userPrincipalDto.UserId, chatId); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking participant")
		return err
	} else if !participant {
		GetLogEntry(c.Request().Context()).Infof("User %v is not participant of chat %v, skipping", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}

	userIds, err := mc.db.GetParticipantsRead(chatId, messageId, size, offset)
	if err != nil {
		return err
	}

	count, err := mc.db.GetParticipantsReadCount(chatId, messageId)
	if err != nil {
		return err
	}

	message, ownerId, err := mc.db.GetMessageBasic(chatId, messageId)
	if err != nil {
		return err
	}

	usersToGet := map[int64]bool{}
	for _, u := range userIds {
		usersToGet[u] = true
	}
	usersToGet[*ownerId] = true

	users, err := mc.restClient.GetUsers(utils.SetToArray(usersToGet), c.Request().Context())
	if err != nil {
		return err
	}

	usersToReturn := []*dto.User{}
	var anOwnerLogin string

	for _, us := range users {
		if utils.Contains(userIds, us.Id) {
			usersToReturn = append(usersToReturn, us)
		}
		if us.Id == userPrincipalDto.UserId {
			anOwnerLogin = us.Login
		}
	}

	preview := createMessagePreview(mc.stripAllTags, *message, anOwnerLogin)


	return c.JSON(http.StatusOK, &MessageReadResponse{
		ParticipantsWrapper: ParticipantsWrapper{
			Data:  usersToReturn,
			Count: count,
		},
		Text:                preview,
	})
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

	preview := createMessagePreview(mc.stripAllTags, bindTo.Text, userPrincipalDto.UserLogin)
	if preview == loginPrefix(userPrincipalDto.UserLogin) {
		preview = ""
	}

	mc.notificator.NotifyAboutMessageBroadcast(c, chatId, userPrincipalDto.UserId, userPrincipalDto.UserLogin, preview)
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
	messageId, hasMessageId, err := mc.db.SetFileItemUuidToNull(userId, chatId, fileItemUuid)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Unable to set FileItemUuid to full for fileItemUuid=%v, chatId=%v", fileItemUuid, chatId)
		return c.NoContent(http.StatusInternalServerError)
	}

	// notifying
	if hasMessageId {
		ids, err := mc.db.GetAllParticipantIds(chatId)
		if err != nil {
			return err
		}
		message, err := getMessage(c, mc.db, mc.restClient, chatId, messageId, userId)
		if err != nil {
			return err
		}
		mc.notificator.NotifyAboutEditMessage(c, ids, chatId, message)
	}

	return c.NoContent(http.StatusOK)
}

func (mc *MessageHandler) PinMessage(c echo.Context) error {
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

	pin, err := GetQueryParamAsBoolean(c, "pin")
	if err != nil {
		return err
	}

	err = db.Transact(mc.db, func(tx *db.Tx) error {
		isParticipant, err := tx.IsParticipant(userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !isParticipant {
			msg := "user " + utils.Int64ToString(userPrincipalDto.UserId) + " is not belongs to chat " + utils.Int64ToString(chatId)
			GetLogEntry(c.Request().Context()).Warnf(msg)
			return c.JSON(http.StatusAccepted, &utils.H{"message": msg})
		}

		participantIds, err := tx.GetAllParticipantIds(chatId)
		if err != nil {
			return err
		}

		if pin {
			err = tx.PinMessage(chatId, messageId, pin)
			if err != nil {
				return err
			}
			err = tx.UnpromoteMessages(chatId)
			if err != nil {
				return err
			}
			err = tx.PromoteMessage(chatId, messageId)
			if err != nil {
				return err
			}

			res, err := getMessage(c, tx, mc.restClient, chatId, messageId, userPrincipalDto.UserId)

			mc.notificator.NotifyAboutEditMessage(c, participantIds, chatId, res)

			count0, err := tx.GetPinnedMessagesCount(chatId)
			if err != nil {
				return err
			}

			// notify about newly promoted result (promoted can be different)
			err = mc.sendPromotePinnedMessageEvent(c, res, tx, chatId, participantIds, userPrincipalDto.UserId, true, count0)
			if err != nil {
				return err
			}
		} else {
			previouslyPromoted, err := tx.GetPinnedPromoted(chatId)
			if err != nil {
				return err
			}

			err = tx.PinMessage(chatId, messageId, pin)
			if err != nil {
				return err
			}

			res, err := getMessage(c, tx, mc.restClient, chatId, messageId, userPrincipalDto.UserId)
			mc.notificator.NotifyAboutEditMessage(c, participantIds, chatId, res)

			count1, err := tx.GetPinnedMessagesCount(chatId)
			if err != nil {
				return err
			}

			// actually instead of unpromote - remove is better
			err = mc.sendPromotePinnedMessageEvent(c, res, tx, chatId, participantIds, userPrincipalDto.UserId, false, count1)
			if err != nil {
				return err
			}

			// promote the previous
			if previouslyPromoted != nil && previouslyPromoted.Id == messageId {
				err = tx.PromotePreviousMessage(chatId)
				if err != nil {
					return err
				}
				promoted, err := tx.GetPinnedPromoted(chatId)
				if err != nil {
					return err
				}
				if promoted != nil {
					count2, err := tx.GetPinnedMessagesCount(chatId)
					if err != nil {
						return err
					}

					res2, err := getMessage(c, tx, mc.restClient, chatId, promoted.Id, userPrincipalDto.UserId)

					err = mc.sendPromotePinnedMessageEvent(c, res2, tx, chatId, participantIds, userPrincipalDto.UserId, true, count2)
					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}

func (mc *MessageHandler) MakeBlogPost(c echo.Context) error {
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

	return db.Transact(mc.db, func(tx *db.Tx) error {
		_, ownerId, err := tx.GetMessageBasic(chatId, messageId)
		if err != nil {
			return err
		}

		if ownerId == nil {
			return c.NoContent(http.StatusNoContent)
		}

		if *ownerId != userPrincipalDto.UserId {
			return c.NoContent(http.StatusUnauthorized)
		}

		prevBlogPostMessageId, err := tx.GetBlogPostMessageId(chatId)
		if err != nil {
			return err
		}

		err = tx.SetBlogPost(chatId, messageId)
		if err != nil {
			return err
		}

		participantIds, err := tx.GetAllParticipantIds(chatId)
		if err != nil {
			return err
		}

		// send edit for previous message - it lost "blog_post == true"
		if prevBlogPostMessageId != nil {
			res0, err := getMessage(c, tx, mc.restClient, chatId, *prevBlogPostMessageId, userPrincipalDto.UserId)
			if err != nil {
				return err
			}
			mc.notificator.NotifyAboutEditMessage(c, participantIds, chatId, res0)
		}
		// notify about new "blog_post == true"
		res, err := getMessage(c, tx, mc.restClient, chatId, messageId, userPrincipalDto.UserId)
		if err != nil {
			return err
		}
		mc.notificator.NotifyAboutEditMessage(c, participantIds, chatId, res)

		return c.JSON(http.StatusOK, res)
	})
}

type MessageWrapper struct {
	Data  []*dto.DisplayMessageDto `json:"data"`
	Count int64                    `json:"totalCount"` // total message number for this user
}

func (mc *MessageHandler) GetPinnedMessages(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	page := utils.FixPageString(c.QueryParam("page"))
	size := utils.FixSizeString(c.QueryParam("size"))
	offset := utils.GetOffset(page, size)

	return db.Transact(mc.db, func(tx *db.Tx) error {
		isParticipant, err := tx.IsParticipant(userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !isParticipant {
			msg := "user " + utils.Int64ToString(userPrincipalDto.UserId) + " is not belongs to chat " + utils.Int64ToString(chatId)
			GetLogEntry(c.Request().Context()).Warnf(msg)
			return c.NoContent(http.StatusUnauthorized)
		}

		messages, err := tx.GetPinnedMessages(chatId, size, offset)
		if err != nil {
			return err
		}

		var ownersSet = map[int64]bool{}
		var chatsPreSet = map[int64]bool{}
		for _, message := range messages {
			populateSets(message, ownersSet, chatsPreSet)
		}
		chatsSet, err := mc.db.GetChatsBasic(chatsPreSet, userPrincipalDto.UserId)
		if err != nil {
			return err
		}
		var owners = getUsersRemotelyOrEmpty(ownersSet, mc.restClient, c)
		messageDtos := make([]*dto.DisplayMessageDto, 0)
		for _, c := range messages {
			converted := convertToMessageDto(c, owners, chatsSet, userPrincipalDto.UserId)

			patchForView(mc.stripAllTags, converted, c.PinPromoted)
			messageDtos = append(messageDtos, converted)
		}

		count, err := tx.GetPinnedMessagesCount(chatId)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, MessageWrapper{
			Data:  messageDtos,
			Count: count,
		})
	})
}

func (mc *MessageHandler) GetPinnedPromotedMessage(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	return db.Transact(mc.db, func(tx *db.Tx) error {
		isParticipant, err := tx.IsParticipant(userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !isParticipant {
			msg := "user " + utils.Int64ToString(userPrincipalDto.UserId) + " is not belongs to chat " + utils.Int64ToString(chatId)
			GetLogEntry(c.Request().Context()).Warnf(msg)
			return c.NoContent(http.StatusUnauthorized)
		}

		message, err := tx.GetPinnedPromoted(chatId)
		if err != nil {
			return err
		}

		if message == nil {
			return c.NoContent(http.StatusNoContent)
		}
		var ownersSet = map[int64]bool{}
		var chatsPreSet = map[int64]bool{}
		populateSets(message, ownersSet, chatsPreSet)

		chatsSet, err := tx.GetChatsBasic(chatsPreSet, userPrincipalDto.UserId)
		if err != nil {
			return err
		}

		var owners = getUsersRemotelyOrEmpty(ownersSet, mc.restClient, c)
		res := convertToMessageDto(message, owners, chatsSet, userPrincipalDto.UserId)
		patchForView(mc.stripAllTags, res, true)

		return c.JSON(http.StatusOK, res)
	})
}

func patchForView(cleanTagsPolicy *services.StripTagsPolicy, message *dto.DisplayMessageDto, promote bool) {
	if message.EmbedMessage != nil && message.EmbedMessage.EmbedType == dto.EmbedMessageTypeResend {
		message.Text = message.EmbedMessage.Text
	}
	message.Text = createMessagePreviewWithoutLogin(cleanTagsPolicy, message.Text)
	message.PinnedPromoted = &promote
}

func (mc *MessageHandler) sendPromotePinnedMessageEvent(c echo.Context, displayMessage *dto.DisplayMessageDto, tx *db.Tx, chatId int64, participantIds []int64, behalfUserId int64, promote bool, count int64) error {
	var copiedMsg = &dto.DisplayMessageDto{}

	err := deepcopy.Copy(copiedMsg, displayMessage)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("error during performing deep copy message: %s", err)
		return err
	}

	patchForView(mc.stripAllTags, copiedMsg, promote)

	// notify about promote to the pinned
	mc.notificator.NotifyAboutPromotePinnedMessage(c, chatId, &dto.PinnedMessageEvent{
		Message:    *copiedMsg,
		TotalCount: count,
	}, promote, participantIds)
	return nil
}

func (mc *MessageHandler) findMentions(messageText string, isFindingNewMentions bool, users map[int64]*dto.User, userOnlines map[int64]*dto.UserOnline) ([]int64, string) {
	var aMap = map[int64]bool{}
	withoutSourceTags := mc.stripSourceContent.Sanitize(messageText)
	for _, user := range users {
		if strings.Contains(withoutSourceTags, "@"+allUsers) && isFindingNewMentions {
			aMap[user.Id] = true
		} else if strings.Contains(withoutSourceTags, "@"+user.Login) {
			aMap[user.Id] = true
		}
	}
	for _, user := range userOnlines {
		if strings.Contains(withoutSourceTags, "@"+hereUsers) && isFindingNewMentions {
			aMap[user.Id] = true
		}
	}

	withoutAnyHtml := mc.stripAllTags.Sanitize(withoutSourceTags)
	if withoutAnyHtml != "" {
		withoutAnyHtml = createMessagePreviewWithoutLogin(mc.stripAllTags, withoutAnyHtml)
	}

	return utils.SetToArray(aMap), withoutAnyHtml
}

func (mc *MessageHandler) wasReplyAdded(oldMessage *db.Message, messageRendered *dto.DisplayMessageDto, chatId int64) (*dto.ReplyDto, *int64) {
	var replyWasMissed = true
	if oldMessage != nil && oldMessage.ResponseEmbeddedMessageType != nil && *oldMessage.ResponseEmbeddedMessageType == dto.EmbedMessageTypeReply && oldMessage.ResponseEmbeddedMessageReplyId != nil {
		replyWasMissed = false
	}
	if replyWasMissed && messageRendered.EmbedMessage != nil && messageRendered.EmbedMessage.Owner != nil && messageRendered.Owner != nil && messageRendered.EmbedMessage.EmbedType == dto.EmbedMessageTypeReply {

		withoutAnyHtml := createMessagePreviewWithoutLogin(mc.stripAllTags, messageRendered.Text)

		return &dto.ReplyDto{
			MessageId:        messageRendered.Id,
			ChatId:           chatId,
			ReplyableMessage: withoutAnyHtml,
		}, &messageRendered.EmbedMessage.Owner.Id
	} else {
		return nil, nil
	}
}

func (mc *MessageHandler) wasReplyRemoved(oldMessage *db.Message, messageRendered *dto.DisplayMessageDto, chatId int64) (*dto.ReplyDto, *int64) {
	var replyWasPresented = true
	if oldMessage.ResponseEmbeddedMessageType != nil && *oldMessage.ResponseEmbeddedMessageType == dto.EmbedMessageTypeReply && oldMessage.ResponseEmbeddedMessageReplyId == nil {
		replyWasPresented = false
	}
	if replyWasPresented && ((messageRendered != nil && messageRendered.EmbedMessage == nil) || (messageRendered == nil)) {
		return &dto.ReplyDto{
			MessageId: oldMessage.Id,
			ChatId:    chatId,
		}, oldMessage.ResponseEmbeddedMessageReplyOwnerId
	} else {
		return nil, nil
	}
}

// in order to be able to see video in chrome after minio link's ttl expiration
func patchStorageUrlToPreventCachingVideo(text string) string {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		Logger.Warnf("Unagle to read html: %v", err)
		return ""
	}

	wlArr := []string{"", viper.GetString("baseUrl")}

	doc.Find("video").Each(func(i int, s *goquery.Selection) {
		maybeVideo := s.First()
		if maybeVideo != nil {
			src, srcExists := maybeVideo.Attr("src")
			if srcExists && utils.ContainsUrl(wlArr, src) {
				newurl, err := addTimeToUrl(src)
				if err != nil {
					Logger.Warnf("Unagle to change url: %v", err)
					return
				}
				maybeVideo.SetAttr("src", newurl)
			}
		}
	})

	ret, err := doc.Find("html").Find("body").Html()
	if err != nil {
		Logger.Warnf("Unagle to write html: %v", err)
		return ""
	}
	return ret
}

func addTimeToUrl(src string) (string, error) {
	parsed, err := url.Parse(src)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	query.Set("time", utils.Int64ToString(time.Now().Unix()))
	parsed.RawQuery = query.Encode()

	newurl := parsed.String()
	return newurl, nil
}
