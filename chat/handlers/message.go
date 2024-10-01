package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

const maxDisplayableUsers = 10

type EditMessageDto struct {
	Id int64 `json:"id"`
	CreateMessageDto
}

type CreateMessageDto struct {
	Text                string                   `json:"text"`
	BlogPost            bool                     `json:"blogPost"`
	FileItemUuid        *string                  `json:"fileItemUuid"`
	EmbedMessageRequest *dto.EmbedMessageRequest `json:"embedMessage"`
}

type MessageHandler struct {
	db                 *db.DB
	policy             *services.SanitizerPolicy
	stripSourceContent *services.StripSourcePolicy
	stripAllTags       *services.StripTagsPolicy
	notificator        *services.Events
	restClient         *client.RestClient
}

func NewMessageHandler(dbR *db.DB, policy *services.SanitizerPolicy, stripSourceContent *services.StripSourcePolicy, stripAllTags *services.StripTagsPolicy, notificator *services.Events, restClient *client.RestClient) *MessageHandler {
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

	isParticipant, err := mc.db.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !isParticipant {
		return c.NoContent(http.StatusUnauthorized)
	}

	fileItemUuid := c.Param("fileItemUuid")
	messageId, err := mc.db.FindMessageByFileItemUuid(c.Request().Context(), chatId, fileItemUuid)
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

	return db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		isParticipant, err := tx.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !isParticipant {
			return c.NoContent(http.StatusNoContent)
		}

		if messages, err := tx.GetMessages(c.Request().Context(), chatId, size, startingFromItemId, reverse, hasHash, searchString); err != nil {
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
				populateSets(message, ownersSet, chatsPreSet, true)
			}
			chatsSet, err := tx.GetChatsBasic(c.Request().Context(), chatsPreSet, userPrincipalDto.UserId)
			if err != nil {
				return err
			}
			var users = getUsersRemotelyOrEmpty(c.Request().Context(), ownersSet, mc.restClient)
			areAdminsMap, err := getAreAdmins(c.Request().Context(), tx, users, chatId)
			if err != nil {
				return err
			}

			messageDtos := make([]*dto.DisplayMessageDto, 0)
			for _, mm := range messages {
				messageDtos = append(messageDtos, convertToMessageDto(c.Request().Context(), mm, users, chatsSet, userPrincipalDto.UserId, areAdminsMap[userPrincipalDto.UserId]))
			}

			GetLogEntry(c.Request().Context()).Infof("Successfully returning %v messages", len(messageDtos))
			return c.JSON(http.StatusOK, messageDtos)
		}
	})
}

func getMessage(c echo.Context, co db.CommonOperations, restClient *client.RestClient, chatId int64, messageId int64, behalfUserId int64, behalfUserIsAdminInChat bool) (*dto.DisplayMessageDto, error) {
	message, chatsSet, users, err := prepareDataForMessage(c.Request().Context(), co, restClient, chatId, messageId, behalfUserId)

	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error get messages from db %v", err)
		return nil, err
	}

	if message == nil {
		return nil, nil
	}

	return convertToMessageDto(c.Request().Context(), message, users, chatsSet, behalfUserId, behalfUserIsAdminInChat), nil
}

func prepareDataForMessage(ctx context.Context, co db.CommonOperations, restClient *client.RestClient, chatId int64, messageId int64, behalfUserId int64) (*db.Message, map[int64]*db.BasicChatDtoExtended, map[int64]*dto.User, error) {
	if message, err := co.GetMessage(ctx, chatId, behalfUserId, messageId); err != nil {
		GetLogEntry(ctx).Errorf("Error get messages from db %v", err)
		return nil, nil, nil, err
	} else {
		if message == nil {
			return nil, nil, nil, nil
		}
		var ownersSet = map[int64]bool{}
		var chatsPreSet = map[int64]bool{}
		populateSets(message, ownersSet, chatsPreSet, true)

		chatsSet, err := co.GetChatsBasic(ctx, chatsPreSet, behalfUserId)
		if err != nil {
			return nil, nil, nil, err
		}

		var users = getUsersRemotelyOrEmpty(ctx, ownersSet, restClient)
		return message, chatsSet, users, nil
	}
}

func getMessageWithoutPersonalized(ctx context.Context, co db.CommonOperations, restClient *client.RestClient, chatId int64, messageId int64, behalfUserId int64) (*dto.DisplayMessageDto, error) {
	message, chatsSet, users, err := prepareDataForMessage(ctx, co, restClient, chatId, messageId, behalfUserId)

	if err != nil {
		GetLogEntry(ctx).Errorf("Error get messages from db %v", err)
		return nil, err
	}

	if message == nil {
		return nil, nil
	}

	return convertToMessageDtoWithoutPersonalized(ctx, message, users, chatsSet), nil
}

func populateSets(message *db.Message, ownersSet map[int64]bool, chatsPreSet map[int64]bool, countReactions bool) {
	ownersSet[message.OwnerId] = true
	chatsPreSet[message.ChatId] = true
	if message.ResponseEmbeddedMessageReplyOwnerId != nil {
		var embeddedMessageReplyOwnerId = *message.ResponseEmbeddedMessageReplyOwnerId
		ownersSet[embeddedMessageReplyOwnerId] = true
	} else if message.ResponseEmbeddedMessageResendOwnerId != nil {
		var embeddedMessageResendOwnerId = *message.ResponseEmbeddedMessageResendOwnerId
		ownersSet[embeddedMessageResendOwnerId] = true
		var embeddedMessageResendChatId = *message.ResponseEmbeddedMessageResendChatId
		chatsPreSet[embeddedMessageResendChatId] = true
	}

	if countReactions {
		takeOnAccountReactions(ownersSet, message.Reactions)
	}
}

func takeOnAccountReactions(ownersSet map[int64]bool, messageReactions []db.Reaction) {
	var currDisplayableUsers = 0
	for _, r := range messageReactions {

		if !ownersSet[r.UserId] {
			ownersSet[r.UserId] = true
			currDisplayableUsers++
		}

		if currDisplayableUsers >= maxDisplayableUsers {
			break
		}
	}
}

func getOnlyMaxDisplayableUsers(users []int64) []int64 {
	var usersToReturn []int64 = make([]int64, 0)
	var currDisplayableUsers = 0

	for _, u := range users {
		usersToReturn = append(usersToReturn, u)
		currDisplayableUsers++

		if currDisplayableUsers >= maxDisplayableUsers {
			break
		}
	}

	return usersToReturn
}

type ReactionPut struct {
	Reaction string `json:"reaction"`
}

func (mc *MessageHandler) ReactionMessage(c echo.Context) error {
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

	messageIdString := c.Param("messageId")
	messageId, err := utils.ParseInt64(messageIdString)
	if err != nil {
		return err
	}

	return db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		isParticipant, err := tx.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !isParticipant {
			return c.NoContent(http.StatusUnauthorized)
		}

		var bindTo = new(ReactionPut)
		if err := c.Bind(bindTo); err != nil {
			GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
			return err
		}

		wasAdded, err := tx.FlipReaction(c.Request().Context(), userPrincipalDto.UserId, chatId, messageId, bindTo.Reaction)
		if err != nil {
			GetLogEntry(c.Request().Context()).Warnf("Error during flipping reaction %v", err)
			return err
		}

		reactionUserIds, err := tx.GetReactionUsers(c.Request().Context(), chatId, messageId, bindTo.Reaction)
		if err != nil {
			GetLogEntry(c.Request().Context()).Warnf("Error during counting reaction %v", err)
			return err
		}

		var wasChanged bool
		var count = len(reactionUserIds)
		if count > 0 {
			wasChanged = true // false means removed
		}
		maxDisplayableReactionUsers := getOnlyMaxDisplayableUsers(reactionUserIds)
		reactionUserMap := getUsersRemotelyOrEmptyFromSlice(c.Request().Context(), maxDisplayableReactionUsers, mc.restClient)

		reactionUsers := make([]*dto.User, 0)
		for _, userId := range reactionUserIds {
			user := reactionUserMap[userId]
			if user != nil {
				reactionUsers = append(reactionUsers, user)
			} else {
				reactionUsers = append(reactionUsers, getDeletedUser(userId))
			}
		}

		mc.notificator.SendReactionEvent(c.Request().Context(), wasChanged, chatId, messageId, bindTo.Reaction, reactionUsers, count, tx)

		chatNameForNotification, err := mc.getChatNameForNotification(c.Request().Context(), tx, chatId)
		if err != nil {
			return err
		}

		_, messageOwnerId, _, _, err := tx.GetMessageBasic(c.Request().Context(), chatId, messageId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants")
			return err
		}

		// sends notification to the notification microservice
		if *messageOwnerId != userPrincipalDto.UserId {
			mc.notificator.SendReactionOnYourMessage(c.Request().Context(), wasAdded, chatId, messageId, *messageOwnerId, bindTo.Reaction, userPrincipalDto.UserId, userPrincipalDto.UserLogin, userPrincipalDto.Avatar, chatNameForNotification)
		}

		GetLogEntry(c.Request().Context()).Infof("Got reaction %v", bindTo.Reaction)
		return c.NoContent(http.StatusOK)
	})
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

	admin, err := mc.db.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}

	message, err := getMessage(c, mc.db, mc.restClient, chatId, messageId, userPrincipalDto.UserId, admin)
	if err != nil {
		return err
	}
	if message == nil {
		return c.NoContent(http.StatusNotFound)
	}
	GetLogEntry(c.Request().Context()).Infof("Successfully returning message %v", message)
	return c.JSON(http.StatusOK, message)
}

func getDeletedUser(id int64) *dto.User {
	return &dto.User{Login: fmt.Sprintf("deleted_user_%v", id), Id: id}
}

func convertToMessageDto(ctx context.Context, dbMessage *db.Message, users map[int64]*dto.User, chats map[int64]*db.BasicChatDtoExtended, behalfUserId int64, behalfUserIsAdminInChat bool) *dto.DisplayMessageDto {

	ret := convertToMessageDtoWithoutPersonalized(ctx, dbMessage, users, chats)

	messageChat, ok := chats[dbMessage.ChatId]
	if !ok {
		GetLogEntry(ctx).Errorf("Unable to get message's chat for message id = %v, chat id = %v", dbMessage.Id, dbMessage.ChatId)
	}
	ret.SetPersonalizedFields(messageChat.RegularParticipantCanPublishMessage, messageChat.RegularParticipantCanPinMessage, behalfUserIsAdminInChat, behalfUserId)

	return ret
}

func convertToMessageDtoWithoutPersonalized(ctx context.Context, dbMessage *db.Message, users map[int64]*dto.User, chats map[int64]*db.BasicChatDtoExtended) *dto.DisplayMessageDto {
	user := users[dbMessage.OwnerId]
	if user == nil {
		user = getDeletedUser(dbMessage.OwnerId)
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
		Published:      dbMessage.Published,
	}
	ret.Text = patchStorageUrlToPreventCachingVideo(ctx, ret.Text)

	if dbMessage.ResponseEmbeddedMessageReplyOwnerId != nil {
		embeddedUser := users[*dbMessage.ResponseEmbeddedMessageReplyOwnerId]
		ret.EmbedMessage = &dto.EmbedMessageResponse{
			Id:        *dbMessage.ResponseEmbeddedMessageReplyId,
			Text:      *dbMessage.ResponseEmbeddedMessageReplyText,
			EmbedType: *dbMessage.ResponseEmbeddedMessageType,
			Owner:     embeddedUser,
		}
	} else if dbMessage.ResponseEmbeddedMessageResendOwnerId != nil {
		embeddedUser := users[*dbMessage.ResponseEmbeddedMessageResendOwnerId]
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

	ret.Reactions = convertReactions(dbMessage.Reactions, users)

	return ret
}

func convertToPublishedMessageDto(cleanTagsPolicy *services.StripTagsPolicy, dbMessage *db.Message, users map[int64]*dto.User) *dto.PublishedMessageDto {
	user := users[dbMessage.OwnerId]
	if user == nil {
		user = getDeletedUser(dbMessage.OwnerId)
	}
	ret := &dto.PublishedMessageDto{
		Id:             dbMessage.Id,
		Text:           dbMessage.Text,
		ChatId:         dbMessage.ChatId,
		OwnerId:        dbMessage.OwnerId,
		Owner:          user,
		CreateDateTime: dbMessage.CreateDateTime,
	}

	ret.Text = createMessagePreviewWithoutLogin(cleanTagsPolicy, ret.Text)

	return ret
}

func convertToPinnedMessageDto(cleanTagsPolicy *services.StripTagsPolicy, dbMessage *db.Message, users map[int64]*dto.User) *dto.PinnedMessageDto {
	user := users[dbMessage.OwnerId]
	if user == nil {
		user = getDeletedUser(dbMessage.OwnerId)
	}
	ret := &dto.PinnedMessageDto{
		Id:             dbMessage.Id,
		Text:           dbMessage.Text,
		ChatId:         dbMessage.ChatId,
		OwnerId:        dbMessage.OwnerId,
		Owner:          user,
		PinnedPromoted: dbMessage.PinPromoted,
		CreateDateTime: dbMessage.CreateDateTime,
	}

	ret.Text = createMessagePreviewWithoutLogin(cleanTagsPolicy, ret.Text)

	return ret
}

func convertReactions(dbReactionsOfMessage []db.Reaction, users map[int64]*dto.User) []dto.Reaction {
	var convertedReactionsOfMessageToReturn = make([]dto.Reaction, 0)

	for _, dbReaction := range dbReactionsOfMessage {
		user := users[dbReaction.UserId]
		wasSummed := false
		for j, existingReaction := range convertedReactionsOfMessageToReturn {
			if dbReaction.Reaction == existingReaction.Reaction {
				convertedReactionsOfMessageToReturn[j].Count = existingReaction.Count + 1

				usersOfThisReaction := convertedReactionsOfMessageToReturn[j].Users
				if user != nil {
					usersOfThisReaction = append(usersOfThisReaction, user)
				} else {
					usersOfThisReaction = append(usersOfThisReaction, getDeletedUser(dbReaction.UserId))
				}

				convertedReactionsOfMessageToReturn[j].Users = usersOfThisReaction

				wasSummed = true
			}
		}
		if !wasSummed {
			usersOfThisReaction := []*dto.User{}
			if user != nil {
				usersOfThisReaction = append(usersOfThisReaction, user)
			} else {
				usersOfThisReaction = append(usersOfThisReaction, getDeletedUser(dbReaction.UserId))
			}

			convertedReactionsOfMessageToReturn = append(convertedReactionsOfMessageToReturn, dto.Reaction{
				Count:    1,
				Reaction: dbReaction.Reaction,
				Users:    usersOfThisReaction,
			})
		}
	}

	return convertedReactionsOfMessageToReturn
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

type notParticipantError struct{}

func (m *notParticipantError) Error() string {
	return "You are not a participant"
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

	messageId, errOuter := db.TransactWithResult(c.Request().Context(), mc.db, func(tx *db.Tx) (int64, error) {
		if participant, err := tx.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
			return 0, err
		} else if !participant {
			return 0, &notParticipantError{}
		}
		creatableMessage, err := convertToCreatableMessage(c.Request().Context(), bindTo, userPrincipalDto, chatId, mc.policy)
		if err != nil {
			return 0, err
		}

		err = mc.validateAndSetEmbedFieldsEmbedMessage(c.Request().Context(), tx, bindTo, creatableMessage)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during checking embed %v", err)
			return 0, err
		}

		chatBasic, err := tx.GetChatBasic(c.Request().Context(), chatId)
		if err != nil {
			return 0, err
		}

		if chatBasic.IsBlog {
			hasMessages, err := tx.HasMessages(c.Request().Context(), chatId)
			if err != nil {
				return 0, err
			}

			if !hasMessages {
				creatableMessage.BlogPost = true
			}
		}

		messageId, _, _, err := tx.CreateMessage(c.Request().Context(), creatableMessage)
		if err != nil {
			return 0, err
		}
		_, err = tx.AddMessageRead(c.Request().Context(), messageId, userPrincipalDto.UserId, chatId) // not to send to myself (1/2)
		if err != nil {
			return 0, err
		}
		if tx.UpdateChatLastDatetimeChat(c.Request().Context(), chatId) != nil {
			return 0, err
		}
		return messageId, nil
	})
	if errOuter != nil {
		var mediaError *MediaUrlErr
		if errors.As(errOuter, &mediaError) {
			return c.JSON(http.StatusBadRequest, &utils.H{"message": mediaError.Error(), "businessErrorCode": badMediaUrl})
		}
		var npe *notParticipantError
		if errors.As(errOuter, &npe) {
			return c.JSON(http.StatusBadRequest, &utils.H{"message": "You are not allowed to write to this chat"})
		}

		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
		return errOuter
	}

	errOuter = db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		chatDto, err := getChat(c.Request().Context(), tx, mc.restClient, chatId, userPrincipalDto.UserId, 0, 0)
		if err != nil {
			return err
		}

		message, err := getMessageWithoutPersonalized(c.Request().Context(), tx, mc.restClient, chatId, messageId, userPrincipalDto.UserId) // personal values will be set inside IterateOverChatParticipantIds -> event.go
		if err != nil {
			return err
		}

		chatNameForNotification, err := mc.getChatNameForNotification(c.Request().Context(), tx, chatId)
		if err != nil {
			return err
		}
		var reply, userToSendTo = mc.wasReplyAdded(nil, message, chatId)
		mc.notificator.NotifyAddReply(c.Request().Context(), reply, userToSendTo, userPrincipalDto.UserId, userPrincipalDto.UserLogin, userPrincipalDto.Avatar, chatNameForNotification)

		messageTextWithoutTags := createMessagePreview(mc.stripAllTags, message.Text, userPrincipalDto.UserLogin)

		err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
			areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, chatId)
			if err != nil {
				return err
			}

			mc.notificator.NotifyAboutChangeChat(c.Request().Context(), chatDto, participantIds, len(chatDto.ParticipantIds) == 1, true, tx, areAdmins)
			shouldSendHasUnreadMessagesMap, err := tx.ShouldSendHasUnreadMessagesCountBatchCommon(c.Request().Context(), chatId, participantIds)
			if err != nil {
				return err
			}
			for participantId, should := range shouldSendHasUnreadMessagesMap {
				if participantId != userPrincipalDto.UserId { // not to send to myself (2/2)
					mc.notificator.NotifyAboutHasNewMessagesChanged(c.Request().Context(), participantId, should)

					meAsUser := dto.User{Id: userPrincipalDto.UserId, Login: userPrincipalDto.UserLogin, Avatar: null.StringFromPtr(userPrincipalDto.Avatar)}
					var sch dto.ChatDtoWithTetATet = &simpleChat{
						Id:        chatDto.Id,
						Name:      chatDto.Name,
						IsTetATet: chatDto.IsTetATet,
						Avatar:    chatDto.Avatar,
					}
					utils.ReplaceChatNameToLoginForTetATet(
						sch,
						&meAsUser,
						participantId,
						false, // because participantId != userPrincipalDto.UserId above
					)
					mc.notificator.NotifyNewMessageBrowserNotification(c.Request().Context(), true, participantId, chatId, sch.GetName(), sch.GetAvatar(), message.Id, messageTextWithoutTags, userPrincipalDto.UserId, userPrincipalDto.UserLogin)
				}
			}
			var users = getUsersRemotelyOrEmptyFromSlice(c.Request().Context(), participantIds, mc.restClient)
			var userOnlines = getUserOnlinesRemotelyOrEmptyFromSlice(c.Request().Context(), participantIds, mc.restClient)
			var addedMentions, strippedText = mc.findMentions(message.Text, true, users, userOnlines)
			var reallyAddedMentions = excludeMyself(addedMentions, userPrincipalDto)
			mc.notificator.NotifyAddMention(c.Request().Context(), reallyAddedMentions, chatId, message.Id, strippedText, userPrincipalDto.UserId, userPrincipalDto.UserLogin, userPrincipalDto.Avatar, chatNameForNotification)
			mc.notificator.NotifyAboutNewMessage(c.Request().Context(), participantIds, chatId, message, toChatBasic(chatDto), areAdmins)
			return nil
		})
		if err != nil {
			return err
		}
		//mc.notificator.ChatNotifyMessageCount(participantIds, c, chatId, tx) - it's included in NotifyAboutChangeChat

		return c.JSON(http.StatusCreated, &utils.H{"id": messageId})
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func toChatBasic(chatDto *dto.ChatDto) *db.BasicChatDto {
	if chatDto == nil {
		return nil
	} else {
		return &db.BasicChatDto{
			Id:                                  chatDto.Id,
			Title:                               chatDto.Name,
			IsTetATet:                           chatDto.IsTetATet,
			CanResend:                           chatDto.CanResend,
			AvailableToSearch:                   chatDto.AvailableToSearch,
			IsBlog:                              chatDto.Blog,
			CreateDateTime:                      chatDto.LastUpdateDateTime, // TODO
			RegularParticipantCanPublishMessage: chatDto.RegularParticipantCanPublishMessage,
			RegularParticipantCanPinMessage:     chatDto.RegularParticipantCanPinMessage,
		}
	}
}

func (mc *MessageHandler) getChatNameForNotification(ctx context.Context, tx *db.Tx, chatId int64) (string, error) {
	chatBasic, err := tx.GetChatBasic(ctx, chatId)
	if err != nil {
		return "", err
	}
	chatName := chatBasic.Title
	if chatBasic.IsTetATet {
		chatName = ""
	}
	return chatName, nil

}

func (mc *MessageHandler) validateAndSetEmbedFieldsEmbedMessage(ctx context.Context, tx *db.Tx, input *CreateMessageDto, receiver *db.Message) error {
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
			chat, err := tx.GetChatBasic(ctx, input.EmbedMessageRequest.ChatId)
			if err != nil {
				return err
			}
			if !chat.CanResend {
				return errors.New("Resending is forbidden for this chat")
			}
			messageText, messageOwnerId, _, _, err := tx.GetMessageBasic(ctx, input.EmbedMessageRequest.ChatId, input.EmbedMessageRequest.Id)
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

func convertToCreatableMessage(ctx context.Context, dto *CreateMessageDto, authPrincipal *auth.AuthResult, chatId int64, policy *services.SanitizerPolicy) (*db.Message, error) {
	trimmedAndSanitized, err := TrimAmdSanitizeMessage(ctx, policy, dto.Text)
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

	errOuter := db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		editableMessage, err := convertToEditableMessage(c.Request().Context(), bindTo, userPrincipalDto, chatId, mc.policy)
		if err != nil {
			return err
		}

		err = mc.validateAndSetEmbedFieldsEmbedMessage(c.Request().Context(), tx, &bindTo.CreateMessageDto, editableMessage)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during checking embed %v", err)
			return err
		}

		oldMessage, err := tx.GetMessage(c.Request().Context(), chatId, userPrincipalDto.UserId, editableMessage.Id)
		if err != nil {
			return err
		}

		if err != nil {
			return err
		}
		err = tx.EditMessage(c.Request().Context(), editableMessage)
		if err != nil {
			return err
		}

		dbMessage, chatsSet, users0, err := prepareDataForMessage(c.Request().Context(), tx, mc.restClient, chatId, bindTo.Id, userPrincipalDto.UserId) // personal values will be set inside IterateOverChatParticipantIds -> event.go

		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error get messages from db %v", err)
			return err
		}

		message := convertToMessageDtoWithoutPersonalized(c.Request().Context(), dbMessage, users0, chatsSet)

		if err != nil {
			return err
		}

		chatNameForNotification, err := mc.getChatNameForNotification(c.Request().Context(), tx, chatId)
		if err != nil {
			return err
		}

		chatBasic, err := tx.GetChatBasic(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		var replyAdded, userToSendToAdded = mc.wasReplyAdded(oldMessage, message, chatId)
		mc.notificator.NotifyAddReply(c.Request().Context(), replyAdded, userToSendToAdded, userPrincipalDto.UserId, userPrincipalDto.UserLogin, userPrincipalDto.Avatar, chatNameForNotification)
		var replyRemoved, userToSendRemoved = mc.wasReplyRemoved(oldMessage, message, chatId)
		mc.notificator.NotifyRemoveReply(c.Request().Context(), replyRemoved, userToSendRemoved)

		count0, err := tx.GetPublishedMessagesCount(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		count1, err := tx.GetPinnedMessagesCount(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
			areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, chatId)
			if err != nil {
				return err
			}

			var users = getUsersRemotelyOrEmptyFromSlice(c.Request().Context(), participantIds, mc.restClient)
			var userOnlines = getUserOnlinesRemotelyOrEmptyFromSlice(c.Request().Context(), participantIds, mc.restClient)
			var oldMentions, _ = mc.findMentions(oldMessage.Text, false, users, userOnlines)

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

			var reallyAddedMentions = excludeMyself(userIdsToNotifyAboutMentionCreated, userPrincipalDto)
			mc.notificator.NotifyAddMention(c.Request().Context(), reallyAddedMentions, chatId, message.Id, strippedText, userPrincipalDto.UserId, userPrincipalDto.UserLogin, userPrincipalDto.Avatar, chatNameForNotification)
			mc.notificator.NotifyRemoveMention(c.Request().Context(), userIdsToNotifyAboutMentionDeleted, chatId, message.Id)

			mc.notificator.NotifyAboutEditMessage(c.Request().Context(), participantIds, chatId, message, chatBasic, areAdmins)

			var copiedMsg = convertToPublishedMessageDto(mc.stripAllTags, dbMessage, users0)
			mc.notificator.NotifyAboutPublishedMessageEdit(c.Request().Context(), chatId, &dto.PublishedMessageEvent{
				Message:    *copiedMsg,
				TotalCount: count0,
			}, participantIds, chatBasic.RegularParticipantCanPublishMessage, areAdmins)

			converted := convertToPinnedMessageDto(mc.stripAllTags, dbMessage, users0)
			for _, participantId := range participantIds {
				converted.CanPin = dto.CanPinMessage(chatBasic.RegularParticipantCanPinMessage, areAdmins[participantId])
				mc.notificator.NotifyAboutPromotePinnedMessageEdit(c.Request().Context(), chatId, &dto.PinnedMessageEvent{
					Message:    *converted,
					TotalCount: count1,
				}, participantId)
			}

			return nil
		})
		if err != nil {
			return err
		}

		return c.JSON(http.StatusCreated, &utils.H{"id": bindTo.Id})
	})
	if errOuter != nil {
		var mediaError *MediaUrlErr
		if errors.As(errOuter, &mediaError) {
			return c.JSON(http.StatusBadRequest, &utils.H{"message": mediaError.Error(), "businessErrorCode": badMediaUrl})
		}

		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
		return errOuter
	}
	return errOuter
}

type MessageFilterDto struct {
	SearchString string `json:"searchString"`
	MessageId    int64  `json:"messageId"`
}

func (mc *MessageHandler) Filter(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var bindTo = new(MessageFilterDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	searchString := TrimAmdSanitize(mc.policy, bindTo.SearchString)

	return db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		participant, err := tx.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !participant {
			return c.JSON(http.StatusBadRequest, &utils.H{"message": "You are not allowed to search in this chat"})
		}
		found, err := tx.MessageFilter(c.Request().Context(), chatId, searchString, bindTo.MessageId)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, &utils.H{"found": found})
	})
}

type SetFileItemUuid struct {
	FileItemUuid *string `json:"fileItemUuid"`
	MessageId    int64   `json:"messageId"`
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

	isParticipant, err := mc.db.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId)
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

	_, ownerId, _, _, err := mc.db.GetMessageBasic(c.Request().Context(), chatId, bindTo.MessageId)
	if err != nil {
		return err
	}
	if ownerId == nil || *ownerId != userPrincipalDto.UserId {
		msg := "user " + utils.Int64ToString(userPrincipalDto.UserId) + " is not owner of message " + utils.Int64ToString(bindTo.MessageId)
		GetLogEntry(c.Request().Context()).Warnf(msg)
		return c.JSON(http.StatusAccepted, &utils.H{"message": msg})
	}

	err = mc.db.SetFileItemUuidTo(c.Request().Context(), userPrincipalDto.UserId, chatId, bindTo.MessageId, bindTo.FileItemUuid)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Unable to set FileItemUuid to full for fileItemUuid=%v, chatId=%v", bindTo.FileItemUuid, chatId)
		return c.NoContent(http.StatusInternalServerError)
	}

	message, err := getMessageWithoutPersonalized(c.Request().Context(), mc.db, mc.restClient, chatId, bindTo.MessageId, userPrincipalDto.UserId) // personal values will be set inside IterateOverChatParticipantIds -> event.go
	if err != nil {
		return err
	}

	chatBasic, err := mc.db.GetChatBasic(c.Request().Context(), chatId)
	if err != nil {
		return err
	}

	// notifying
	err = mc.db.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
		areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), mc.db, participantIds, chatId)
		if err != nil {
			return err
		}

		mc.notificator.NotifyAboutEditMessage(c.Request().Context(), participantIds, chatId, message, chatBasic, areAdmins)
		return nil
	})
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func excludeMyself(mentionedUserIds []int64, principalDto *auth.AuthResult) []int64 {
	return utils.Remove(mentionedUserIds, principalDto.UserId)
}

func convertToEditableMessage(ctx context.Context, dto *EditMessageDto, authPrincipal *auth.AuthResult, chatId int64, policy *services.SanitizerPolicy) (*db.Message, error) {
	trimmedAndSanitized, err := TrimAmdSanitizeMessage(ctx, policy, dto.Text)
	if err != nil {
		return nil, err
	}
	return &db.Message{
		Id:           dto.Id,
		Text:         trimmedAndSanitized,
		ChatId:       chatId,
		OwnerId:      authPrincipal.UserId,
		EditDateTime: null.TimeFrom(time.Now().UTC()),
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
	oldMessage, err := mc.db.GetMessage(c.Request().Context(), chatId, userPrincipalDto.UserId, messageId)
	if err != nil {
		return err
	}

	if err := mc.db.DeleteMessage(c.Request().Context(), messageId, userPrincipalDto.UserId, chatId); err != nil {
		return err
	}

	count0, err := mc.db.GetPublishedMessagesCount(c.Request().Context(), chatId)
	if err != nil {
		return err
	}

	count1, err := mc.db.GetPinnedMessagesCount(c.Request().Context(), chatId)
	if err != nil {
		return err
	}

	err = mc.db.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
		var users = getUsersRemotelyOrEmptyFromSlice(c.Request().Context(), participantIds, mc.restClient)
		var userOnlines = getUserOnlinesRemotelyOrEmptyFromSlice(c.Request().Context(), participantIds, mc.restClient)

		var oldMentions, _ = mc.findMentions(oldMessage.Text, false, users, userOnlines)
		mc.notificator.NotifyRemoveMention(c.Request().Context(), oldMentions, chatId, messageId)

		cd := &dto.DisplayMessageDto{
			Id:     messageId,
			ChatId: chatId,
		}
		mc.notificator.NotifyAboutDeleteMessage(c.Request().Context(), participantIds, chatId, cd)

		mc.notificator.NotifyAboutPublishedMessage(c.Request().Context(), chatId, &dto.PublishedMessageEvent{
			Message: dto.PublishedMessageDto{
				Id:             messageId,
				ChatId:         chatId,
				CreateDateTime: oldMessage.CreateDateTime,
			},
			TotalCount: count0,
		}, false, participantIds, false, map[int64]bool{})

		for _, participantId := range participantIds {
			mc.notificator.NotifyAboutPromotePinnedMessage(c.Request().Context(), chatId, &dto.PinnedMessageEvent{
				Message: dto.PinnedMessageDto{
					Id:             messageId,
					ChatId:         chatId,
					CreateDateTime: oldMessage.CreateDateTime,
				},
				TotalCount: count1,
			}, false, participantId)
		}

		for _, participantId := range participantIds {
			mc.notificator.NotifyNewMessageBrowserNotification(c.Request().Context(), false, participantId, chatId, "", null.StringFromPtr(nil), messageId, "", userPrincipalDto.UserId, userPrincipalDto.UserLogin)
		}

		return nil
	})
	if err != nil {
		return err
	}

	var replyRemoved, userToSendRemoved = mc.wasReplyRemoved(oldMessage, nil, chatId)
	mc.notificator.NotifyRemoveReply(c.Request().Context(), replyRemoved, userToSendRemoved)

	return c.JSON(http.StatusAccepted, &utils.H{"id": messageId})
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

	return db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		// here we don't check message ownership because user can read foreign messages
		// (any user has their own last read message per chat)

		err := mc.addMessageReadAndSendIt(tx, c, chatId, messageId, userPrincipalDto.UserId)
		if err != nil {
			return err
		}
		mc.notificator.NotifyRemoveMention(c.Request().Context(), []int64{userPrincipalDto.UserId}, chatId, messageId)
		mc.notificator.NotifyRemoveReply(c.Request().Context(), &dto.ReplyDto{
			MessageId: messageId,
			ChatId:    chatId,
		}, &userPrincipalDto.UserId)

		mc.notificator.NotifyNewMessageBrowserNotification(c.Request().Context(), false, userPrincipalDto.UserId, chatId, "", null.StringFromPtr(nil), messageId, "", NonExistentUser, "")

		_, messageOwnerId, _, _, err := tx.GetMessageBasic(c.Request().Context(), chatId, messageId)
		if err != nil {
			return err
		}

		// remove notifications about reactions on my message on mousemove
		if *messageOwnerId == userPrincipalDto.UserId {
			chatNameForNotification, err := mc.getChatNameForNotification(c.Request().Context(), tx, chatId)
			if err != nil {
				return err
			}

			reactions, err := tx.GetReactionsOnMessage(c.Request().Context(), chatId, messageId)
			if err != nil {
				return err
			}

			for _, reaction := range reactions {
				mc.notificator.SendReactionOnYourMessage(c.Request().Context(), false, chatId, messageId, *messageOwnerId, reaction.Reaction, userPrincipalDto.UserId, userPrincipalDto.UserLogin, userPrincipalDto.Avatar, chatNameForNotification)
			}
		}

		return c.NoContent(http.StatusAccepted)
	})
}

func (mc *MessageHandler) addMessageReadAndSendIt(tx *db.Tx, c echo.Context, chatId int64, messageId int64, userId int64) error {
	_, err := tx.AddMessageRead(c.Request().Context(), messageId, userId, chatId)
	if err != nil {
		return err
	}
	mc.notificator.ChatNotifyMessageCount(c.Request().Context(), []int64{userId}, chatId, tx)

	has, err := tx.HasUnreadMessages(c.Request().Context(), userId)
	if err != nil {
		return err
	}
	mc.notificator.NotifyAboutHasNewMessagesChanged(c.Request().Context(), userId, has)

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

	GetLogEntry(c.Request().Context()).Debugf("Processing GetReadMessageUsers user %v, chatId %v, messageId %v", userPrincipalDto.UserId, chatId, messageId)

	if participant, err := mc.db.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking participant")
		return err
	} else if !participant {
		GetLogEntry(c.Request().Context()).Infof("User %v is not participant of chat %v, skipping", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusUnauthorized)
	}

	userIds, err := mc.db.GetParticipantsRead(c.Request().Context(), chatId, messageId, size, offset)
	if err != nil {
		return err
	}

	count, err := mc.db.GetParticipantsReadCount(c.Request().Context(), chatId, messageId)
	if err != nil {
		return err
	}

	message, ownerId, _, _, err := mc.db.GetMessageBasic(c.Request().Context(), chatId, messageId)
	if err != nil {
		return err
	}

	usersToGet := map[int64]bool{}
	for _, u := range userIds {
		usersToGet[u] = true
	}
	usersToGet[*ownerId] = true

	users, err := mc.restClient.GetUsers(c.Request().Context(), utils.SetToArray(usersToGet))
	if err != nil {
		return err
	}

	usersToReturn := []*dto.User{}
	var anOwnerLogin string

	for _, us := range users {
		if utils.Contains(userIds, us.Id) {
			usersToReturn = append(usersToReturn, us)
		}
		if us.Id == *ownerId {
			anOwnerLogin = us.Login
		}
	}

	preview := createMessagePreview(mc.stripAllTags, *message, anOwnerLogin)

	return c.JSON(http.StatusOK, &MessageReadResponse{
		ParticipantsWrapper: ParticipantsWrapper{
			Data:  usersToReturn,
			Count: count,
		},
		Text: preview,
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

	if participant, err := mc.db.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during checking participant")
		return err
	} else if !participant {
		GetLogEntry(c.Request().Context()).Infof("User %v is not participant of chat %v, skipping", userPrincipalDto.UserId, chatId)
		return c.NoContent(http.StatusAccepted)
	}

	var ownersSet = map[int64]bool{}
	ownersSet[userPrincipalDto.UserId] = true
	var owners = getUsersRemotelyOrEmpty(c.Request().Context(), ownersSet, mc.restClient)
	typingUser := owners[userPrincipalDto.UserId]

	mc.notificator.NotifyAboutMessageTyping(c.Request().Context(), chatId, typingUser, mc.db)
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

	if participant, err := mc.db.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId); err != nil {
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

	mc.notificator.NotifyAboutMessageBroadcast(c.Request().Context(), chatId, userPrincipalDto.UserId, userPrincipalDto.UserLogin, preview, mc.db)
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
	isParticipant, err := mc.db.IsParticipant(c.Request().Context(), userId, chatId)
	if err != nil {
		return err
	}
	if !isParticipant {
		msg := "user " + c.QueryParam("userId") + " is not belongs to chat " + c.QueryParam("chatId")
		GetLogEntry(c.Request().Context()).Warnf(msg)
		return c.JSON(http.StatusAccepted, &utils.H{"message": msg})
	}
	fileItemUuid := c.QueryParam("fileItemUuid")
	messageId, hasMessageId, err := mc.db.SetFileItemUuidToNull(c.Request().Context(), userId, chatId, fileItemUuid)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Unable to set FileItemUuid to full for fileItemUuid=%v, chatId=%v", fileItemUuid, chatId)
		return c.NoContent(http.StatusInternalServerError)
	}

	// notifying
	if hasMessageId {
		message, err := getMessageWithoutPersonalized(c.Request().Context(), mc.db, mc.restClient, chatId, messageId, userId) // personal values will be set inside IterateOverChatParticipantIds -> event.go
		if err != nil {
			return err
		}
		chatBasic, err := mc.db.GetChatBasic(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		err = mc.db.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
			areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), mc.db, participantIds, chatId)
			if err != nil {
				return err
			}

			mc.notificator.NotifyAboutEditMessage(c.Request().Context(), participantIds, chatId, message, chatBasic, areAdmins)
			return nil
		})
		if err != nil {
			return err
		}
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

	err = db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		isParticipant, err := tx.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !isParticipant {
			msg := "user " + utils.Int64ToString(userPrincipalDto.UserId) + " is not belongs to chat " + utils.Int64ToString(chatId)
			GetLogEntry(c.Request().Context()).Warnf(msg)
			return c.JSON(http.StatusAccepted, &utils.H{"message": msg})
		}

		isAdmin, err := tx.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}

		if pin {
			err = tx.PinMessage(c.Request().Context(), chatId, messageId, pin)
			if err != nil {
				return err
			}
			err = tx.UnpromoteMessages(c.Request().Context(), chatId)
			if err != nil {
				return err
			}
			err = tx.PromoteMessage(c.Request().Context(), chatId, messageId)
			if err != nil {
				return err
			}

			message, chatsSet, users, err := prepareDataForMessage(c.Request().Context(), tx, mc.restClient, chatId, messageId, userPrincipalDto.UserId)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error get messages from db %v", err)
				return err
			}
			chatBasic := chatsSet[message.ChatId].BasicChatDto
			if !dto.CanPinMessage(chatBasic.RegularParticipantCanPinMessage, isAdmin) {
				return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You cannot pin messages in this chat"})
			}

			res := convertToMessageDtoWithoutPersonalized(c.Request().Context(), message, users, chatsSet) // personal values will be set inside IterateOverChatParticipantIds -> event.go

			count0, err := tx.GetPinnedMessagesCount(c.Request().Context(), chatId)
			if err != nil {
				return err
			}

			err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
				areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, chatId)
				if err != nil {
					return err
				}
				mc.notificator.NotifyAboutEditMessage(c.Request().Context(), participantIds, chatId, res, &chatBasic, areAdmins)

				// notify about newly promoted result (promoted can be different)
				errInternal := mc.sendPromotePinnedMessageEvent(c.Request().Context(), &chatBasic, isAdmin, mc.stripAllTags, message, users, chatId, participantIds, userPrincipalDto.UserId, true, count0)
				return errInternal
			})
			if err != nil {
				return err
			}
		} else {
			previouslyPromoted, err := tx.GetPinnedPromoted(c.Request().Context(), chatId)
			if err != nil {
				return err
			}

			err = tx.PinMessage(c.Request().Context(), chatId, messageId, pin)
			if err != nil {
				return err
			}

			message, chatsSet, users, err := prepareDataForMessage(c.Request().Context(), tx, mc.restClient, chatId, messageId, userPrincipalDto.UserId)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error get messages from db %v", err)
				return err
			}
			chatBasic := chatsSet[message.ChatId].BasicChatDto
			if !dto.CanPinMessage(chatBasic.RegularParticipantCanPinMessage, isAdmin) {
				return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You cannot unpin messages in this chat"})
			}

			res := convertToMessageDtoWithoutPersonalized(c.Request().Context(), message, users, chatsSet) // personal values will be set inside IterateOverChatParticipantIds -> event.go

			count1, err := tx.GetPinnedMessagesCount(c.Request().Context(), chatId)
			if err != nil {
				return err
			}

			err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
				areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, chatId)
				if err != nil {
					return err
				}

				mc.notificator.NotifyAboutEditMessage(c.Request().Context(), participantIds, chatId, res, &chatBasic, areAdmins)

				// actually instead of unpromote - remove is better
				errInternal := mc.sendPromotePinnedMessageEvent(c.Request().Context(), &chatBasic, isAdmin, mc.stripAllTags, message, users, chatId, participantIds, userPrincipalDto.UserId, false, count1)
				return errInternal
			})
			if err != nil {
				return err
			}

			// promote the previous
			if previouslyPromoted != nil && previouslyPromoted.Id == messageId {
				err = tx.PromotePreviousMessage(c.Request().Context(), chatId)
				if err != nil {
					return err
				}
				promoted, err := tx.GetPinnedPromoted(c.Request().Context(), chatId)
				if err != nil {
					return err
				}
				if promoted != nil {
					count2, err := tx.GetPinnedMessagesCount(c.Request().Context(), chatId)
					if err != nil {
						return err
					}

					message2, _, users2, err := prepareDataForMessage(c.Request().Context(), tx, mc.restClient, chatId, promoted.Id, userPrincipalDto.UserId)
					if err != nil {
						GetLogEntry(c.Request().Context()).Errorf("Error get messages from db %v", err)
						return err
					}

					err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
						errInternal := mc.sendPromotePinnedMessageEvent(c.Request().Context(), &chatBasic, isAdmin, mc.stripAllTags, message2, users2, chatId, participantIds, userPrincipalDto.UserId, true, count2)
						return errInternal
					})
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

func (mc *MessageHandler) PublishMessage(c echo.Context) error {
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

	publish, err := GetQueryParamAsBoolean(c, "publish")
	if err != nil {
		return err
	}

	return db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		isParticipant, err := tx.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !isParticipant {
			msg := "user " + utils.Int64ToString(userPrincipalDto.UserId) + " is not belongs to chat " + utils.Int64ToString(chatId)
			GetLogEntry(c.Request().Context()).Warnf(msg)
			return c.JSON(http.StatusUnauthorized, &utils.H{"message": msg})
		}

		chatBasic, err := tx.GetChatBasic(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		isAdmin, err := tx.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}

		_, ownerId, _, _, err := tx.GetMessageBasic(c.Request().Context(), chatId, messageId)
		if err != nil {
			return err
		}
		if ownerId == nil {
			return c.NoContent(http.StatusNoContent)
		}

		if !dto.CanPublishMessage(chatBasic.RegularParticipantCanPublishMessage, isAdmin, *ownerId, userPrincipalDto.UserId) {
			return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You cannot publish messages in this chat"})
		}

		err = tx.PublishMessage(c.Request().Context(), chatId, messageId, publish)
		if err != nil {
			return err
		}

		message, chatsSet, users, err := prepareDataForMessage(c.Request().Context(), tx, mc.restClient, chatId, messageId, userPrincipalDto.UserId)

		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error get messages from db %v", err)
			return err
		}

		res := convertToMessageDtoWithoutPersonalized(c.Request().Context(), message, users, chatsSet) // personal values will be set inside IterateOverChatParticipantIds -> event.go

		count0, err := tx.GetPublishedMessagesCount(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
			areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, chatId)
			if err != nil {
				return err
			}

			mc.notificator.NotifyAboutEditMessage(c.Request().Context(), participantIds, chatId, res, chatBasic, areAdmins)

			var copiedMsg = convertToPublishedMessageDto(mc.stripAllTags, message, users)

			mc.notificator.NotifyAboutPublishedMessage(c.Request().Context(), chatId, &dto.PublishedMessageEvent{
				Message:    *copiedMsg,
				TotalCount: count0,
			}, publish, participantIds, chatBasic.RegularParticipantCanPublishMessage, areAdmins)
			return nil
		})
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	})
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

	return db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		_, ownerId, _, _, err := tx.GetMessageBasic(c.Request().Context(), chatId, messageId)
		if err != nil {
			return err
		}

		if ownerId == nil {
			return c.NoContent(http.StatusNoContent)
		}

		if *ownerId != userPrincipalDto.UserId {
			return c.NoContent(http.StatusUnauthorized)
		}

		prevBlogPostMessageId, err := tx.GetBlogPostMessageId(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		err = tx.SetBlogPost(c.Request().Context(), chatId, messageId)
		if err != nil {
			return err
		}

		// send edit for previous message - it lost "blog_post == true"
		var res0 *dto.DisplayMessageDto
		if prevBlogPostMessageId != nil {
			res0, err = getMessageWithoutPersonalized(c.Request().Context(), tx, mc.restClient, chatId, *prevBlogPostMessageId, userPrincipalDto.UserId) // personal values will be set inside IterateOverChatParticipantIds -> event.go
			if err != nil {
				return err
			}
		}
		// notify about new "blog_post == true"
		res, err := getMessageWithoutPersonalized(c.Request().Context(), tx, mc.restClient, chatId, messageId, userPrincipalDto.UserId) // personal values will be set inside IterateOverChatParticipantIds -> event.go
		if err != nil {
			return err
		}

		chatBasic, err := tx.GetChatBasic(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		err = tx.IterateOverChatParticipantIds(c.Request().Context(), chatId, func(participantIds []int64) error {
			areAdmins, err := getAreAdminsOfUserIds(c.Request().Context(), tx, participantIds, chatId)
			if err != nil {
				return err
			}
			if res0 != nil {
				mc.notificator.NotifyAboutEditMessage(c.Request().Context(), participantIds, chatId, res0, chatBasic, areAdmins)
			}
			mc.notificator.NotifyAboutEditMessage(c.Request().Context(), participantIds, chatId, res, chatBasic, areAdmins)
			return nil
		})
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	})
}

type PinnedMessagesWrapper struct {
	Data  []*dto.PinnedMessageDto `json:"items"`
	Count int64                   `json:"count"` // total pinned messages number
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

	return db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		isParticipant, err := tx.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !isParticipant {
			msg := "user " + utils.Int64ToString(userPrincipalDto.UserId) + " is not belongs to chat " + utils.Int64ToString(chatId)
			GetLogEntry(c.Request().Context()).Warnf(msg)
			return c.NoContent(http.StatusUnauthorized)
		}

		isAdmin, err := tx.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}

		messages, err := tx.GetPinnedMessages(c.Request().Context(), chatId, size, offset)
		if err != nil {
			return err
		}

		var ownersSet = map[int64]bool{}
		var chatsPreSet = map[int64]bool{}
		for _, message := range messages {
			populateSets(message, ownersSet, chatsPreSet, false)
		}
		var owners = getUsersRemotelyOrEmpty(c.Request().Context(), ownersSet, mc.restClient)
		messageDtos := make([]*dto.PinnedMessageDto, 0)
		chatBasic, err := tx.GetChatBasic(c.Request().Context(), chatId)
		if err != nil {
			return err
		}
		for _, message := range messages {

			converted := convertToPinnedMessageDto(mc.stripAllTags, message, owners)
			converted.CanPin = dto.CanPinMessage(chatBasic.RegularParticipantCanPinMessage, isAdmin)

			messageDtos = append(messageDtos, converted)
		}

		count, err := tx.GetPinnedMessagesCount(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, PinnedMessagesWrapper{
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

	return db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		isParticipant, err := tx.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !isParticipant {
			msg := "user " + utils.Int64ToString(userPrincipalDto.UserId) + " is not belongs to chat " + utils.Int64ToString(chatId)
			GetLogEntry(c.Request().Context()).Warnf(msg)
			return c.NoContent(http.StatusUnauthorized)
		}

		message, err := tx.GetPinnedPromoted(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		if message == nil {
			return c.NoContent(http.StatusNoContent)
		}
		var ownersSet = map[int64]bool{}
		var chatsPreSet = map[int64]bool{}
		populateSets(message, ownersSet, chatsPreSet, false)

		chatsSet, err := tx.GetChatsBasic(c.Request().Context(), chatsPreSet, userPrincipalDto.UserId)
		if err != nil {
			return err
		}

		var owners = getUsersRemotelyOrEmpty(c.Request().Context(), ownersSet, mc.restClient)
		res := convertToMessageDtoWithoutPersonalized(c.Request().Context(), message, owners, chatsSet) // the actual personal values don't needed here
		patchForViewAndSetPromoted(mc.stripAllTags, res, true)

		return c.JSON(http.StatusOK, res)
	})
}

type PublishedMessagesWrapper struct {
	Data  []*dto.PublishedMessageDto `json:"items"`
	Count int64                      `json:"count"` // total pinned messages number
}

func (mc *MessageHandler) GetPublishedMessages(c echo.Context) error {
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

	return db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {
		isParticipant, err := tx.IsParticipant(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !isParticipant {
			msg := "user " + utils.Int64ToString(userPrincipalDto.UserId) + " is not belongs to chat " + utils.Int64ToString(chatId)
			GetLogEntry(c.Request().Context()).Warnf(msg)
			return c.NoContent(http.StatusUnauthorized)
		}

		admin, err := tx.IsAdmin(c.Request().Context(), userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}

		messages, err := tx.GetPublishedMessages(c.Request().Context(), chatId, size, offset)
		if err != nil {
			return err
		}

		chatBasic, err := tx.GetChatBasic(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		var ownersSet = map[int64]bool{}
		for _, message := range messages {
			ownersSet[message.OwnerId] = true
		}
		var owners = getUsersRemotelyOrEmpty(c.Request().Context(), ownersSet, mc.restClient)
		messageDtos := make([]*dto.PublishedMessageDto, 0)

		for _, message := range messages {

			converted := convertToPublishedMessageDto(mc.stripAllTags, message, owners) // the actual personal values don't needed here
			converted.CanPublish = dto.CanPublishMessage(chatBasic.RegularParticipantCanPublishMessage, admin, message.OwnerId, userPrincipalDto.UserId)

			messageDtos = append(messageDtos, converted)
		}

		count, err := tx.GetPublishedMessagesCount(c.Request().Context(), chatId)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, PublishedMessagesWrapper{
			Data:  messageDtos,
			Count: count,
		})
	})
}

func getAreAdmins(ctx context.Context, co db.CommonOperations, owners map[int64]*dto.User, chatId int64) (map[int64]bool, error) {

	ownerIds := make([]int64, 0)
	for ownerId, _ := range owners {
		ownerIds = append(ownerIds, ownerId)
	}

	return getAreAdminsOfUserIds(ctx, co, ownerIds, chatId)
}

func getAreAdminsOfUserIds(ctx context.Context, co db.CommonOperations, userIds []int64, chatId int64) (map[int64]bool, error) {
	areAdminsMap := map[int64]bool{}

	areAdmins, err := co.IsAdminBatchByParticipants(ctx, userIds, chatId)
	if err != nil {
		return areAdminsMap, err
	}
	for _, areAdmin := range areAdmins {
		areAdminsMap[areAdmin.UserId] = areAdmin.Admin
	}
	return areAdminsMap, nil
}

type PublishedMessageWrapper struct {
	Message *dto.DisplayMessageDto `json:"message"`
	Title   string                 `json:"title"`
	Preview string                 `json:"preview"`
}

func (mc *MessageHandler) GetPublishedMessage(c echo.Context) error {
	chatId, err := utils.ParseInt64(c.Param("id"))
	if err != nil {
		return err
	}

	messageId, err := utils.ParseInt64(c.Param("messageId"))
	if err != nil {
		return err
	}

	return db.Transact(c.Request().Context(), mc.db, func(tx *db.Tx) error {

		chatBasic, err := tx.GetChatBasic(c.Request().Context(), chatId)
		if err != nil {
			return err
		}
		if chatBasic == nil {
			return c.NoContent(http.StatusNoContent)
		}

		message, err := tx.GetMessagePublic(c.Request().Context(), chatId, messageId)
		if err != nil {
			return err
		}
		if message == nil {
			return c.NoContent(http.StatusNoContent)
		}

		var ownersSet = map[int64]bool{}
		var chatsPreSet = map[int64]bool{}
		populateSets(message, ownersSet, chatsPreSet, true)
		participantIds, err := tx.GetParticipantIds(c.Request().Context(), chatId, utils.DefaultSize, utils.DefaultOffset)
		if err != nil {
			return err
		}
		for _, participantId := range participantIds {
			ownersSet[participantId] = true
		}
		chatsSet, err := mc.db.GetChatsBasic(c.Request().Context(), chatsPreSet, NonExistentUser)
		if err != nil {
			return err
		}

		var owners = getUsersRemotelyOrEmpty(c.Request().Context(), ownersSet, mc.restClient)

		convertedMessage := convertToMessageDtoWithoutPersonalized(c.Request().Context(), message, owners, chatsSet) // the actual personal values don't needed here

		convertedMessage.Text = PatchStorageUrlToPublic(c.Request().Context(), convertedMessage.Text, convertedMessage.Id)

		preview := stripTagsAndCut(mc.stripAllTags, viper.GetInt("previewMaxTextSize"), convertedMessage.Text)

		aTitle := chatBasic.Title
		if chatBasic.IsTetATet {
			first := true
			aTitle = ""
			for _, user := range owners {
				if !first {
					aTitle += ", "
				}
				aTitle += user.Login

				first = false
			}
		}

		return c.JSON(http.StatusOK, PublishedMessageWrapper{
			Message: convertedMessage,
			Title:   aTitle,
			Preview: preview,
		})
	})
}

func patchForView(cleanTagsPolicy *services.StripTagsPolicy, message *dto.DisplayMessageDto) {
	if message.EmbedMessage != nil && message.EmbedMessage.EmbedType == dto.EmbedMessageTypeResend {
		message.Text = message.EmbedMessage.Text
	}
	message.Text = createMessagePreviewWithoutLogin(cleanTagsPolicy, message.Text)
}

func patchForViewAndSetPromoted(cleanTagsPolicy *services.StripTagsPolicy, message *dto.DisplayMessageDto, promote bool) {
	patchForView(cleanTagsPolicy, message)
	message.PinnedPromoted = &promote
}

func (mc *MessageHandler) sendPromotePinnedMessageEvent(ctx context.Context, chatBasic *db.BasicChatDto, isAdmin bool, cleanTagsPolicy *services.StripTagsPolicy, dbMessage *db.Message, users map[int64]*dto.User, chatId int64, participantIds []int64, behalfUserId int64, promote bool, count int64) error {

	messageDto := convertToPinnedMessageDto(cleanTagsPolicy, dbMessage, users)

	for _, participantId := range participantIds {
		messageDto.CanPin = dto.CanPinMessage(chatBasic.RegularParticipantCanPinMessage, isAdmin)

		// notify about promote to the pinned
		mc.notificator.NotifyAboutPromotePinnedMessage(ctx, chatId, &dto.PinnedMessageEvent{
			Message:    *messageDto,
			TotalCount: count,
		}, promote, participantId)
	}
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

// in order to be able to see video in Chrome after minio link's ttl expiration
// see also blog.go :: PatchStorageUrlToPublic
func patchStorageUrlToPreventCachingVideo(ctx context.Context, text string) string {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		GetLogEntry(ctx).Warnf("Unagle to read html: %v", err)
		return ""
	}

	wlArr := []string{"", viper.GetString("frontendUrl")} // if our own server (storage)

	doc.Find("video").Each(func(i int, s *goquery.Selection) {
		maybeVideo := s.First()
		if maybeVideo != nil {
			src, srcExists := maybeVideo.Attr("src")
			if srcExists && utils.ContainsUrl(wlArr, src) {
				newurl, err := addTimeToUrl(src)
				if err != nil {
					GetLogEntry(ctx).Warnf("Unagle to change url: %v", err)
					return
				}
				maybeVideo.SetAttr("src", newurl)
			}
		}
	})

	ret, err := doc.Find("html").Find("body").Html()
	if err != nil {
		GetLogEntry(ctx).Warnf("Unagle to write html: %v", err)
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
	addTimeToUrlValues(&query)
	parsed.RawQuery = query.Encode()

	newurl := parsed.String()
	return newurl, nil
}

func addTimeToUrlValues(query *url.Values) {
	query.Set("time", utils.Int64ToString(time.Now().UTC().Unix()))
}
