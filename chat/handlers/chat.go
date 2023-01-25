package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/getlantern/deepcopy"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
	"nkonev.name/chat/utils"
	"strings"
)

type ChatWrapper struct {
	Data  []*dto.ChatDto `json:"data"`
	Count int64          `json:"totalCount"` // total chat number for this user
}

type ParticipantsWrapper struct {
	Data  []*dto.UserWithAdmin `json:"participants"`
	Count int                  `json:"participantsCount"` // for paginating purposes
}

type EditChatDto struct {
	Id int64 `json:"id"`
	CreateChatDto
}

type CreateChatDto struct {
	Name           string      `json:"name"`
	ParticipantIds *[]int64    `json:"participantIds"`
	Avatar         null.String `json:"avatar"`
	AvatarBig      null.String `json:"avatarBig"`
	CanResend      bool        `json:"canResend"`
}

type ChatHandler struct {
	db              *db.DB
	notificator     services.Events
	restClient      *client.RestClient
	policy          *services.SanitizerPolicy
	cleanTagsPolicy *services.StripTagsPolicy
}

func NewChatHandler(dbR *db.DB, notificator services.Events, restClient *client.RestClient, policy *services.SanitizerPolicy, cleanTagsPolicy *services.StripTagsPolicy) *ChatHandler {
	return &ChatHandler{db: dbR, notificator: notificator, restClient: restClient, policy: policy, cleanTagsPolicy: cleanTagsPolicy}
}

func (a *CreateChatDto) Validate() error {
	return validation.ValidateStruct(a, validation.Field(&a.Name, validation.Required, validation.Length(1, 256)))
}

func (a *EditChatDto) Validate() error {
	return validation.ValidateStruct(a, validation.Field(&a.Name, validation.Required, validation.Length(1, 256)), validation.Field(&a.Id, validation.Required))
}

func (ch *ChatHandler) GetChats(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	page := utils.FixPageString(c.QueryParam("page"))
	size := utils.FixSizeString(c.QueryParam("size"))
	offset := utils.GetOffset(page, size)

	searchString := c.QueryParam("searchString")
	searchString = strings.TrimSpace(searchString)
	var dbChats []*db.ChatWithParticipants
	var additionalFoundUserIds = []int64{}

	if searchString != "" {
		searchString = TrimAmdSanitize(ch.policy, searchString)

		users, _, err := ch.restClient.SearchGetUsers(searchString, true, []int64{}, 0, 0, c.Request().Context())
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error get users from aaa %v", err)
			return err
		}
		for _, u := range users {
			additionalFoundUserIds = append(additionalFoundUserIds, u.Id)
		}
	}

	dbChats, err := ch.db.GetChatsWithParticipants(userPrincipalDto.UserId, size, offset, searchString, additionalFoundUserIds, userPrincipalDto, 0, 0)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error get chats from db %v", err)
		return err
	}

	chatDtos := make([]*dto.ChatDto, 0)
	for _, cc := range dbChats {
		messages, err := ch.db.GetUnreadMessagesCount(cc.Id, userPrincipalDto.UserId)
		if err != nil {
			return err
		}
		cd := convertToDto(cc, []*dto.User{}, messages)
		chatDtos = append(chatDtos, cd)
	}

	var participantIdSet = map[int64]bool{}
	for _, chatDto := range chatDtos {
		for _, participantId := range chatDto.ParticipantIds {
			participantIdSet[participantId] = true
		}
	}
	var users = getUsersRemotelyOrEmpty(participantIdSet, ch.restClient, c)
	for _, chatDto := range chatDtos {
		for _, participantId := range chatDto.ParticipantIds {
			user := users[participantId]
			if user != nil {
				chatDto.Participants = append(chatDto.Participants, user)
				utils.ReplaceChatNameToLoginForTetATet(chatDto, user, userPrincipalDto.UserId)
			}
		}
	}

	userChatCount, err := ch.db.CountChatsPerUser(userPrincipalDto.UserId)
	if err != nil {
		return errors.New("Error during getting user chat count")
	}
	GetLogEntry(c.Request().Context()).Infof("Successfully returning %v chats", len(chatDtos))
	return c.JSON(http.StatusOK, ChatWrapper{Data: chatDtos, Count: userChatCount})
}

func getChat(
	dbR db.CommonOperations,
	restClient *client.RestClient,
	c echo.Context,
	chatId int64,
	behalfParticipantId int64,
	authResult *auth.AuthResult,
	participantsSize, participantsOffset int,
) (*dto.ChatDto, error) {
	fixedParticipantsSize := utils.FixSize(participantsSize)

	var users []*dto.User = []*dto.User{}

	cc, err := dbR.GetChatWithParticipants(behalfParticipantId, chatId, fixedParticipantsSize, participantsOffset)
	if err != nil {
		return nil, err
	}
	if cc == nil {
		return nil, nil
	}

	users, err = restClient.GetUsers(cc.ParticipantsIds, c.Request().Context())
	if err != nil {
		users = []*dto.User{}
		GetLogEntry(c.Request().Context()).Warn("Error during getting users from aaa")
	}

	unreadMessages, err := dbR.GetUnreadMessagesCount(cc.Id, behalfParticipantId)
	if err != nil {
		return nil, err
	}
	chatDto := convertToDto(cc, users, unreadMessages)

	for _, participant := range users {
		utils.ReplaceChatNameToLoginForTetATet(chatDto, participant, behalfParticipantId)
	}

	return chatDto, nil
}

func (ch *ChatHandler) GetChat(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	participantsPage := utils.FixPage(0)
	participantsSize := utils.FixSize(0)
	participantsOffset := utils.GetOffset(participantsPage, participantsSize)

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	if chat, err := getChat(ch.db, ch.restClient, c, chatId, userPrincipalDto.UserId, userPrincipalDto, participantsSize, participantsOffset); err != nil {
		return err
	} else {
		if chat == nil {
			return c.NoContent(http.StatusNotFound)
		} else {
			copiedChat, err := ch.getChatWithAdminedUsers(c, chat, ch.db)
			if err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}

			GetLogEntry(c.Request().Context()).Infof("Successfully returning %v chat", copiedChat)
			return c.JSON(http.StatusOK, copiedChat)
		}
	}
}

func (ch *ChatHandler) getChatWithAdminedUsers(c echo.Context, chat *dto.ChatDto, commonDbOperations db.CommonOperations) (*dto.ChatDtoWithAdmin, error) {
	var copiedChat = &dto.ChatDtoWithAdmin{}
	err := deepcopy.Copy(copiedChat, chat)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("error during performing deep copy chat: %s", err)
		return nil, err
	}

	var adminedUsers []*dto.UserWithAdmin
	for _, participant := range copiedChat.Participants {
		var copied = &dto.UserWithAdmin{}
		if err := deepcopy.Copy(copied, participant); err != nil {
			GetLogEntry(c.Request().Context()).Errorf("error during performing deep copy user: %s", err)
		} else {
			if admin, err := commonDbOperations.IsAdmin(participant.Id, copiedChat.Id); err != nil {
				GetLogEntry(c.Request().Context()).Warnf("Unable to get IsAdmin for user %v in chat %v from db", participant.Id, copiedChat.Id)
			} else {
				copied.Admin = admin
			}

			adminedUsers = append(adminedUsers, copied)
		}
	}
	copiedChat.Participants = adminedUsers
	return copiedChat, nil
}

func convertToDto(c *db.ChatWithParticipants, users []*dto.User, unreadMessages int64) *dto.ChatDto {
	b := dto.BaseChatDto{
		Id:             c.Id,
		Name:           c.Title,
		ParticipantIds: c.ParticipantsIds,
		Avatar:         c.Avatar,
		AvatarBig:      c.AvatarBig,
		IsTetATet:      c.TetATet,
		CanResend:      c.CanResend,

		// see also services/events.go:75 chatNotifyCommon()

		ParticipantsCount:  c.ParticipantsCount,
		LastUpdateDateTime: c.LastUpdateDateTime,
	}

	b.SetPersonalizedFields(c.IsAdmin, unreadMessages)

	return &dto.ChatDto{
		BaseChatDto:  b,
		Participants: users,
	}
}

func (ch *ChatHandler) CreateChat(c echo.Context) error {
	var bindTo = new(CreateChatDto)
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

	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {
		id, _, err := tx.CreateChat(convertToCreatableChat(bindTo, ch.policy))
		if err != nil {
			return err
		}
		// add admin
		if err := tx.AddParticipant(userPrincipalDto.UserId, id, true); err != nil {
			return err
		}

		if bindTo.ParticipantIds != nil {
			participantIds := *bindTo.ParticipantIds
			// add other participants except admin
			for _, participantId := range participantIds {
				if participantId == userPrincipalDto.UserId {
					continue
				}
				if err := tx.AddParticipant(participantId, id, false); err != nil {
					return err
				}
			}
		}

		responseDto, err := getChat(tx, ch.restClient, c, id, userPrincipalDto.UserId, userPrincipalDto, 0, 0)
		if err != nil {
			return err
		}
		copiedChat, err := ch.getChatWithAdminedUsers(c, responseDto, tx)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}

		ch.notificator.NotifyAboutNewChat(c, copiedChat, responseDto.ParticipantIds, tx)
		return c.JSON(http.StatusCreated, responseDto)
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func convertToCreatableChat(d *CreateChatDto, policy *services.SanitizerPolicy) *db.Chat {
	return &db.Chat{
		Title:     TrimAmdSanitize(policy, d.Name),
		CanResend: d.CanResend,
	}
}

func (ch *ChatHandler) DeleteChat(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {
		if admin, err := tx.IsAdmin(userPrincipalDto.UserId, chatId); err != nil {
			return err
		} else if !admin {
			return errors.New(fmt.Sprintf("User %v is not admin of chat %v", userPrincipalDto.UserId, chatId))
		}
		ids, err := tx.GetAllParticipantIds(chatId)
		if err != nil {
			return err
		}
		if err := tx.DeleteChat(chatId); err != nil {
			return err
		}

		ch.notificator.NotifyAboutDeleteChat(c, chatId, ids, tx)
		return c.JSON(http.StatusAccepted, &utils.H{"id": chatId})
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch *ChatHandler) EditChat(c echo.Context) error {
	var bindTo = new(EditChatDto)
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

	var userIdsToNotifyAboutChatChanged []int64
	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {
		if admin, err := tx.IsAdmin(userPrincipalDto.UserId, bindTo.Id); err != nil {
			return err
		} else if !admin {
			return errors.New(fmt.Sprintf("User %v is not admin of chat %v", userPrincipalDto.UserId, bindTo.Id))
		}
		_, err := tx.EditChat(bindTo.Id, TrimAmdSanitize(ch.policy, bindTo.Name), bindTo.Avatar, bindTo.AvatarBig, bindTo.CanResend)
		if err != nil {
			return err
		}

		if ids, err := tx.GetAllParticipantIds(bindTo.Id); err != nil {
			return err
		} else {
			for _, pid := range ids {
				userIdsToNotifyAboutChatChanged = append(userIdsToNotifyAboutChatChanged, pid)
			}
		}

		if responseDto, err := getChat(tx, ch.restClient, c, bindTo.Id, userPrincipalDto.UserId, userPrincipalDto, 0, 0); err != nil {
			return err
		} else {
			copiedChat, err := ch.getChatWithAdminedUsers(c, responseDto, tx)
			if err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}

			ch.notificator.NotifyAboutChangeChat(c, copiedChat, userIdsToNotifyAboutChatChanged, tx)
			return c.JSON(http.StatusAccepted, responseDto)
		}
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch *ChatHandler) LeaveChat(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {
		if err := tx.DeleteParticipant(userPrincipalDto.UserId, chatId); err != nil {
			return err
		}

		firstUser, err := tx.GetFirstParticipant(chatId)
		if err != nil {
			return err
		}
		if responseDto, err := getChat(tx, ch.restClient, c, chatId, firstUser, nil, 0, 0); err != nil {
			return err
		} else {
			copiedChat, err := ch.getChatWithAdminedUsers(c, responseDto, tx)
			if err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}
			ch.notificator.NotifyAboutDeleteParticipants(c, responseDto.ParticipantIds, chatId, []int64{userPrincipalDto.UserId})

			ch.notificator.NotifyAboutChangeChat(c, copiedChat, responseDto.ParticipantIds, tx)
			ch.notificator.NotifyAboutDeleteChat(c, copiedChat.Id, []int64{userPrincipalDto.UserId}, tx)
			return c.JSON(http.StatusAccepted, responseDto)
		}
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

type ChangeAdminResponseDto struct {
	Admin bool `json:"admin"`
}

func (ch *ChatHandler) ChangeParticipant(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {

		// check that I am admin
		admin, err := tx.IsAdmin(userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !admin {
			return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
		}
		interestingUserId, err := GetPathParamAsInt64(c, "participantId")
		participant, err := tx.IsParticipant(interestingUserId, chatId)
		if err != nil {
			return err
		}
		if !participant {
			return c.JSON(http.StatusBadRequest, &utils.H{"message": "User is not belong to chat"})
		}

		newAdmin, err := GetQueryParamAsBoolean(c, "admin")
		if err != nil {
			return err
		}
		err = tx.SetAdmin(interestingUserId, chatId, newAdmin)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during changing chat admin in database %v", err)
			return err
		}
		participantIds, err := tx.GetAllParticipantIds(chatId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants %v", err)
			return err
		}

		newUsersWithAdmin, err := ch.getParticipantsWithAdmin(tx, []int64{interestingUserId}, chatId, c.Request().Context())
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting participants aith admin %v", err)
			return err
		}

		ch.notificator.NotifyAboutChangeParticipants(c, participantIds, chatId, newUsersWithAdmin)

		tmpDto, err := getChat(tx, ch.restClient, c, chatId, userPrincipalDto.UserId, nil, 0, 0)
		if err != nil {
			return err
		}
		copiedChat, err := ch.getChatWithAdminedUsers(c, tmpDto, tx)
		if err != nil {
			return err
		}
		ch.notificator.NotifyAboutChangeChat(c, copiedChat, []int64{interestingUserId}, tx)

		return c.JSON(http.StatusAccepted, newUsersWithAdmin)
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch *ChatHandler) DeleteParticipant(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {

		// check that I am admin
		admin, err := tx.IsAdmin(userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !admin {
			return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
		}
		interestingUserId, err := GetPathParamAsInt64(c, "participantId")

		err = tx.DeleteParticipant(interestingUserId, chatId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during changing chat admin in database %v", err)
			return err
		}
		participantIds, err := tx.GetAllParticipantIds(chatId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants %v", err)
			return err
		}

		ch.notificator.NotifyAboutDeleteParticipants(c, participantIds, chatId, []int64{interestingUserId})

		tmpDto, err := getChat(tx, ch.restClient, c, chatId, userPrincipalDto.UserId, nil, 0, 0)
		if err != nil {
			return err
		}
		copiedChat, err := ch.getChatWithAdminedUsers(c, tmpDto, tx)
		if err != nil {
			return err
		}
		ch.notificator.NotifyAboutChangeChat(c, copiedChat, tmpDto.ParticipantIds, tx)

		ch.notificator.NotifyAboutDeleteChat(c, chatId, []int64{interestingUserId}, tx)

		return c.NoContent(http.StatusAccepted)
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch *ChatHandler) getParticipantsWithAdmin(cdo db.CommonOperations, participantIds []int64, chatId int64, ctx context.Context) ([]*dto.UserWithAdmin, error) {
	newUsers, err := ch.restClient.GetUsers(participantIds, ctx)
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during getting users %v", err)
		return nil, err
	}
	return ch.enrichWithAdmin(cdo, newUsers, chatId, ctx)
}

func (ch *ChatHandler) enrichWithAdmin(cdo db.CommonOperations, users []*dto.User, chatId int64, ctx context.Context) ([]*dto.UserWithAdmin, error) {
	newUsersWithAdmin := []*dto.UserWithAdmin{}
	for _, newUser := range users {
		isAdmin, err := cdo.IsAdmin(newUser.Id, chatId)
		if err != nil {
			GetLogEntry(ctx).Errorf("Error during isAdmin %v", err)
			return nil, err
		}

		u := newUser
		newUsersWithAdmin = append(newUsersWithAdmin, &dto.UserWithAdmin{
			User:  *u,
			Admin: isAdmin,
		})
	}
	return newUsersWithAdmin, nil
}

type AddParticipantsDto struct {
	ParticipantIds []int64 `json:"addParticipantIds"`
}

func (ch *ChatHandler) AddParticipants(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {

		// check that I am admin
		admin, err := tx.IsAdmin(userPrincipalDto.UserId, chatId)
		if err != nil {
			return err
		}
		if !admin {
			return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
		}
		var bindTo = new(AddParticipantsDto)
		if err := c.Bind(bindTo); err != nil {
			GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
			return err
		}

		for _, participantId := range bindTo.ParticipantIds {
			err = tx.AddParticipant(participantId, chatId, false)
			if err != nil {
				GetLogEntry(c.Request().Context()).Errorf("Error during changing chat admin in database %v", err)
				return err
			}
		}
		participantIds, err := tx.GetAllParticipantIds(chatId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting chat participants %v", err)
			return err
		}

		newUsersWithAdmin, err := ch.getParticipantsWithAdmin(tx, bindTo.ParticipantIds, chatId, c.Request().Context())
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting participants aith admin %v", err)
			return err
		}
		ch.notificator.NotifyAboutNewParticipants(c, participantIds, chatId, newUsersWithAdmin)

		tmpDto, err := getChat(tx, ch.restClient, c, chatId, userPrincipalDto.UserId, nil, 0, 0)
		if err != nil {
			return err
		}
		copiedChat, err := ch.getChatWithAdminedUsers(c, tmpDto, tx)
		if err != nil {
			return err
		}
		ch.notificator.NotifyAboutChangeChat(c, copiedChat, tmpDto.ParticipantIds, tx)

		return c.NoContent(http.StatusAccepted)
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch *ChatHandler) SearchForUsersToAdd(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	admin, err := ch.db.IsAdmin(userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !admin {
		return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no access to this chat"})
	}

	searchString := c.QueryParam("searchString")
	excludingIds, err := ch.db.GetAllParticipantIds(chatId)
	if err != nil {
		return err
	}

	users, _, err := ch.restClient.SearchGetUsers(searchString, false, excludingIds, 0, 0, c.Request().Context())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, users)
}

func (ch *ChatHandler) GetParticipants(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	userSearchString := c.QueryParam("searchString")
	userSearchString = strings.TrimSpace(userSearchString)

	participantsPage := utils.FixPageString(c.QueryParam("page"))
	participantsSize := utils.FixSizeString(c.QueryParam("size"))
	participantsOffset := utils.GetOffset(participantsPage, participantsSize)

	var usersWithAdmin []*dto.UserWithAdmin = []*dto.UserWithAdmin{}

	totalFoundUserCount := 0

	if userSearchString != "" {
		allParticipantIds, err := ch.db.GetAllParticipantIds(chatId)
		if err != nil {
			return err
		}
		var users []*dto.User
		users, totalFoundUserCount, err = ch.restClient.SearchGetUsers(userSearchString, true, allParticipantIds, participantsPage, participantsSize, c.Request().Context())
		if err != nil {
			return err
		}
		usersWithAdmin, err = ch.enrichWithAdmin(ch.db, users, chatId, c.Request().Context())
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting participants with admin %v", err)
			return err
		}
	} else {
		participantIds, err := ch.db.GetParticipantIds(chatId, participantsSize, participantsOffset)
		if err != nil {
			return err
		}

		usersWithAdmin, err = ch.getParticipantsWithAdmin(ch.db, participantIds, chatId, c.Request().Context())
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during getting participants with admin %v", err)
			return err
		}

		count, err := ch.db.GetParticipantsCount(chatId)
		if err != nil {
			return err
		}
		totalFoundUserCount = count
	}

	return c.JSON(http.StatusOK, &ParticipantsWrapper{
		Data:  usersWithAdmin,
		Count: totalFoundUserCount,
	})
}

func (ch *ChatHandler) SearchForUsersToMention(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	participant, err := ch.db.IsParticipant(userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !participant {
		return c.NoContent(http.StatusUnauthorized)
	}

	searchString := c.QueryParam("searchString")
	includingIds, err := ch.db.GetAllParticipantIds(chatId)

	users, _, err := ch.restClient.SearchGetUsers(searchString, true, includingIds, 0, 0, c.Request().Context())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, users)
}

func (ch *ChatHandler) CheckAccess(c echo.Context) error {
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}
	userId, err := GetQueryParamAsInt64(c, "userId")
	if err != nil {
		return err
	}
	useCanResend, _ := GetQueryParamAsBoolean(c, "considerCanResend")
	participant, err := ch.db.IsParticipant(userId, chatId)
	if err != nil {
		return err
	}

	if participant {
		return c.NoContent(http.StatusOK)
	} else {
		if useCanResend {
			chat, err := ch.db.GetChatBasic(chatId)
			if err != nil {
				return err
			}
			if chat == nil {
				return c.NoContent(http.StatusUnauthorized)
			}
			if chat.CanResend {
				return c.NoContent(http.StatusOK)
			} else {
				return c.NoContent(http.StatusUnauthorized)
			}
		}
	}
	return c.NoContent(http.StatusUnauthorized)
}

func (ch *ChatHandler) IsAdmin(c echo.Context) error {
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}
	userId, err := GetQueryParamAsInt64(c, "userId")
	if err != nil {
		return err
	}
	isAdmin, err := ch.db.IsAdmin(userId, chatId)
	if err != nil {
		return err
	}
	if isAdmin {
		return c.NoContent(http.StatusOK)
	} else {
		return c.NoContent(http.StatusUnauthorized)
	}
}

type TetATetResponse struct {
	Id int64 `json:"id"`
}

func (ch *ChatHandler) TetATet(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	toParticipantId, err := GetPathParamAsInt64(c, "participantId")
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during parsing participantId %v", err)
		return err
	}

	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {
		// check existing tet-a-tet chat
		exists, chatId, err := tx.IsExistsTetATet(userPrincipalDto.UserId, toParticipantId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during checking exists tet-a-tet chat %v", err)
			return err
		}
		if exists {
			return c.JSON(http.StatusAccepted, TetATetResponse{Id: chatId})
		}

		// create tet-a-tet chat
		chatId2, err := tx.CreateTetATetChat(userPrincipalDto.UserId, toParticipantId)
		if err != nil {
			GetLogEntry(c.Request().Context()).Errorf("Error during creating tet-a-tet chat %v", err)
			return err
		}

		if err := tx.AddParticipant(userPrincipalDto.UserId, chatId2, true); err != nil {
			return err
		}
		if err := tx.AddParticipant(toParticipantId, chatId2, true); err != nil {
			return err
		}

		responseDto, err := getChat(tx, ch.restClient, c, chatId2, userPrincipalDto.UserId, userPrincipalDto, 0, 0)
		if err != nil {
			return err
		}
		copiedChat, err := ch.getChatWithAdminedUsers(c, responseDto, tx)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}

		ch.notificator.NotifyAboutNewChat(c, copiedChat, responseDto.ParticipantIds, tx)

		return c.JSON(http.StatusCreated, TetATetResponse{Id: chatId2})
	})
	if errOuter != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

type CleanHtmlTagsDto struct {
	Text string `json:"text"`
}

func (ch *ChatHandler) CleanHtmlTags(c echo.Context) error {
	bindTo := new(CleanHtmlTagsDto)
	err := c.Bind(bindTo)
	if err != nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during unmarshalling %v", err)
		return err
	}
	tmp := ch.cleanTagsPolicy.Sanitize(bindTo.Text)
	size := utils.Min(len(tmp), viper.GetInt("previewMaxTextSize"))
	withoutAnyHtml := tmp[:size]
	response := CleanHtmlTagsDto{
		withoutAnyHtml,
	}
	return c.JSON(http.StatusOK, response)
}

type ChatExists struct {
	Exists bool `json:"exists"`
}

func (ch *ChatHandler) IsExists(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}
	exists, err := ch.db.IsChatExists(chatId)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ChatExists{exists})
}

type simpleChat struct {
	Id        int64
	Name      string
	IsTetATet bool
	Avatar    null.String
}

func (r *simpleChat) GetId() int64 {
	return r.Id
}

func (r *simpleChat) GetName() string {
	return r.Name
}

func (r *simpleChat) GetAvatar() null.String {
	return r.Avatar
}

func (r *simpleChat) SetName(s string) {
	r.Name = s
}

func (r *simpleChat) SetAvatar(s null.String) {
	r.Avatar = s
}

func (r *simpleChat) GetIsTetATet() bool {
	return r.IsTetATet
}

func (ch *ChatHandler) GetNameForInvite(c echo.Context) error {
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}
	behalfUserId, err := GetQueryParamAsInt64(c, "behalfUserId")
	if err != nil {
		return err
	}
	participantIds, err := GetQueryParamsAsInt64Slice(c, "userIds")
	if err != nil {
		return err
	}

	chat, err := ch.db.GetChat(behalfUserId, chatId)
	if err != nil {
		return err
	}

	behalfUsers, err := ch.restClient.GetUsers([]int64{behalfUserId}, c.Request().Context())
	if err != nil {
		return err
	}
	if len(behalfUsers) != 1 {
		GetLogEntry(c.Request().Context()).Errorf("Behalf user is not found")
		return c.NoContent(http.StatusNotFound)
	}
	behalfUserLogin := behalfUsers[0].Login

	users, err := ch.restClient.GetUsers(participantIds, c.Request().Context())
	if err != nil {
		return err
	}

	ret := []dto.ChatName{}

	for _, user := range users {
		meAsUser := dto.User{Id: behalfUserId, Login: behalfUserLogin}
		var sch dto.ChatDtoWithTetATet = &simpleChat{
			Id:        chat.Id,
			Name:      chat.Title,
			IsTetATet: chat.TetATet,
			Avatar:    chat.Avatar,
		}
		utils.ReplaceChatNameToLoginForTetATet(
			sch,
			&meAsUser,
			user.Id,
		)
		ret = append(ret, dto.ChatName{Name: sch.GetName(), Avatar: sch.GetAvatar(), UserId: user.Id})
	}
	return c.JSON(http.StatusOK, ret)
}

func (ch *ChatHandler) RemoveAllParticipants(c echo.Context) error {
	GetLogEntry(c.Request().Context()).Warnf("Removing ALL participants")
	return ch.db.DeleteAllParticipants()
}

func (ch *ChatHandler) GetChatParticipants(c echo.Context) error {
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}
	ids, err := ch.db.GetAllParticipantIds(chatId)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ids)
}

type ParticipantBelongsToChat struct {
	UserId  int64 `json:"userId"`
	Belongs bool  `json:"belongs"`
}

type ParticipantsBelongToChat struct {
	Users []*ParticipantBelongsToChat `json:"users"`
}

func (ch *ChatHandler) DoesParticipantBelongToChat(c echo.Context) error {
	GetLogEntry(c.Request().Context()).Infof("Checking if participant belongs")
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}
	userIds, err := GetQueryParamsAsInt64Slice(c, "userId")
	if err != nil {
		return err
	}

	participantIds, err := ch.db.GetAllParticipantIds(chatId)
	if err != nil {
		return err
	}

	var users = []*ParticipantBelongsToChat{}
	for _, userId := range userIds {
		var belongs = &ParticipantBelongsToChat{
			UserId:  userId,
			Belongs: false,
		}
		if utils.Contains(participantIds, userId) {
			belongs.Belongs = true
		}
		users = append(users, belongs)
	}

	return c.JSON(http.StatusOK, &ParticipantsBelongToChat{Users: users})
}
