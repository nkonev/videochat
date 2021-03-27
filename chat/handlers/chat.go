package handlers

import (
	"errors"
	"fmt"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/notifications"
	"nkonev.name/chat/utils"
	"strings"
)

type ChatWrapper struct {
	Data  []*dto.ChatDto `json:"data"`
	Count int64          `json:"totalCount"` // total chat number for this user
}

type EditChatDto struct {
	Id int64 `json:"id"`
	CreateChatDto
}

type CreateChatDto struct {
	Name           string  `json:"name"`
	ParticipantIds []int64 `json:"participantIds"`
}

type ChatHandler struct {
	db          db.DB
	notificator notifications.Notifications
	restClient  client.RestClient
}

func NewChatHandler(dbR db.DB, notificator notifications.Notifications, restClient client.RestClient) ChatHandler {
	return ChatHandler{db: dbR, notificator: notificator, restClient: restClient}
}

func (a *CreateChatDto) Validate() error {
	return validation.ValidateStruct(a, validation.Field(&a.Name, validation.Required, validation.Length(1, 256)))
}

func (a *EditChatDto) Validate() error {
	return validation.ValidateStruct(a, validation.Field(&a.Name, validation.Required, validation.Length(1, 256)), validation.Field(&a.Id, validation.Required))
}

func (ch ChatHandler) GetChats(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	page := utils.FixPageString(c.QueryParam("page"))
	size := utils.FixSizeString(c.QueryParam("size"))
	offset := utils.GetOffset(page, size)
	searchString := c.QueryParam("searchString")
	searchString = strings.TrimSpace(searchString)

	if dbChats, err := ch.db.GetChatsWithParticipants(userPrincipalDto.UserId, size, offset, searchString, userPrincipalDto); err != nil {
		GetLogEntry(c.Request()).Errorf("Error get chats from db %v", err)
		return err
	} else {
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
				}
			}
		}

		userChatCount, err := ch.db.CountChatsPerUser(userPrincipalDto.UserId)
		if err != nil {
			return errors.New("Error during getting user chat count")
		}
		GetLogEntry(c.Request()).Infof("Successfully returning %v chats", len(chatDtos))
		return c.JSON(200, ChatWrapper{Data: chatDtos, Count: userChatCount})
	}
}

func getChat(dbR db.CommonOperations, restClient client.RestClient, c echo.Context, chatId int64, behalfParticipantId int64, authResult *auth.AuthResult) (*dto.ChatDto, error) {
	if cc, err := dbR.GetChatWithParticipants(behalfParticipantId, chatId); err != nil {
		return nil, err
	} else {
		if cc == nil {
			return nil, nil
		}

		users, err := restClient.GetUsers(cc.ParticipantsIds, c.Request().Context())
		if err != nil {
			users = []*dto.User{}
			GetLogEntry(c.Request()).Warn("Error during getting users from aaa")
		}

		for _, user := range users {
			if admin, err := dbR.IsAdmin(user.Id, cc.Id); err != nil {
				GetLogEntry(c.Request()).Warnf("Unable to get IsAdmin for user %v in chat %v from db", user.Id, cc.Id)
			} else {
				user.Admin = admin
			}
		}

		unreadMessages, err := dbR.GetUnreadMessagesCount(cc.Id, behalfParticipantId)
		if err != nil {
			return nil, err
		}
		chatDto := convertToDto(cc, users, unreadMessages)
		if authResult != nil && authResult.HasRole("ROLE_ADMIN") {
			chatDto.CanBroadcast = true
		}

		return chatDto, nil
	}
}

func (ch ChatHandler) GetChat(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	if chat, err := getChat(&ch.db, ch.restClient, c, chatId, userPrincipalDto.UserId, userPrincipalDto); err != nil {
		return err
	} else {
		if chat == nil {
			return c.NoContent(http.StatusNotFound)
		} else {
			GetLogEntry(c.Request()).Infof("Successfully returning %v chat", chat)
			return c.JSON(200, chat)
		}
	}
}

func convertToDto(c *db.ChatWithParticipants, users []*dto.User, unreadMessages int64) *dto.ChatDto {
	return &dto.ChatDto{
		Id:                 c.Id,
		Name:               c.Title,
		ParticipantIds:     c.ParticipantsIds,
		Participants:       users,
		CanEdit:            null.BoolFrom(c.IsAdmin),
		LastUpdateDateTime: c.LastUpdateDateTime,
		CanLeave:           null.BoolFrom(!c.IsAdmin),
		UnreadMessages:     unreadMessages,
	}
}

func (ch ChatHandler) CreateChat(c echo.Context) error {
	var bindTo = new(CreateChatDto)
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

	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {
		id, _, err := tx.CreateChat(convertToCreatableChat(bindTo))
		if err != nil {
			return err
		}
		// add admin
		if err := tx.AddParticipant(userPrincipalDto.UserId, id, true); err != nil {
			return err
		}
		// add other participants except admin
		for _, participantId := range bindTo.ParticipantIds {
			if participantId == userPrincipalDto.UserId {
				continue
			}
			if err := tx.AddParticipant(participantId, id, false); err != nil {
				return err
			}
		}

		responseDto, err := getChat(tx, ch.restClient, c, id, userPrincipalDto.UserId, userPrincipalDto)
		if err != nil {
			return err
		}

		ch.notificator.NotifyAboutNewChat(c, responseDto, responseDto.ParticipantIds, tx)
		return c.JSON(http.StatusCreated, responseDto)
	})
	if errOuter != nil {
		GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func convertToCreatableChat(d *CreateChatDto) *db.Chat {
	return &db.Chat{
		Title: d.Name,
	}
}

func (ch ChatHandler) DeleteChat(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {
		if admin, err := tx.IsAdmin(userPrincipalDto.UserId, chatId); err != nil {
			return err
		} else if !admin {
			return errors.New(fmt.Sprintf("User %v is not admin of chat %v", userPrincipalDto.UserId, chatId))
		}
		ids, err := tx.GetParticipantIds(chatId)
		if err != nil {
			return err
		}
		if err := tx.DeleteChat(chatId); err != nil {
			return err
		}
		cd := &dto.ChatDto{
			Id: chatId,
		}
		ch.notificator.NotifyAboutDeleteChat(c, cd, ids, tx)
		return c.JSON(http.StatusAccepted, &utils.H{"id": chatId})
	})
	if errOuter != nil {
		GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch ChatHandler) EditChat(c echo.Context) error {
	var bindTo = new(EditChatDto)
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

	var userIdsToNotifyAboutChatCreated []int64
	var userIdsToNotifyAboutChatDeleted []int64
	var userIdsToNotifyAboutChatChanged []int64
	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {
		if admin, err := tx.IsAdmin(userPrincipalDto.UserId, bindTo.Id); err != nil {
			return err
		} else if !admin {
			return errors.New(fmt.Sprintf("User %v is not admin of chat %v", userPrincipalDto.UserId, bindTo.Id))
		}
		_, err := tx.EditChat(bindTo.Id, bindTo.Name)
		if err != nil {
			return err
		}

		existsChatParticipantIdsFromDatabase, err := tx.GetParticipantIds(bindTo.Id)
		if err != nil {
			return err
		}

		for _, participantIdFromRequest := range bindTo.ParticipantIds {
			// not exists in database
			if !utils.Contains(existsChatParticipantIdsFromDatabase, participantIdFromRequest) {
				if err := tx.AddParticipant(participantIdFromRequest, bindTo.Id, false); err != nil {
					return err
				}
				userIdsToNotifyAboutChatCreated = append(userIdsToNotifyAboutChatCreated, participantIdFromRequest)
			} else { // exists in database
				userIdsToNotifyAboutChatChanged = append(userIdsToNotifyAboutChatChanged, participantIdFromRequest)
			}
		}

		for _, participantIdFromDatabase := range existsChatParticipantIdsFromDatabase {
			// not present in request array and not myself
			if !utils.Contains(bindTo.ParticipantIds, participantIdFromDatabase) && participantIdFromDatabase != userPrincipalDto.UserId {
				if err := tx.DeleteParticipant(participantIdFromDatabase, bindTo.Id); err != nil {
					return err
				}
				userIdsToNotifyAboutChatDeleted = append(userIdsToNotifyAboutChatDeleted, participantIdFromDatabase)
			}
		}

		if responseDto, err := getChat(tx, ch.restClient, c, bindTo.Id, userPrincipalDto.UserId, userPrincipalDto); err != nil {
			return err
		} else {
			ch.notificator.NotifyAboutNewChat(c, responseDto, userIdsToNotifyAboutChatCreated, tx)
			ch.notificator.NotifyAboutDeleteChat(c, responseDto, userIdsToNotifyAboutChatDeleted, tx)
			ch.notificator.NotifyAboutChangeChat(c, responseDto, userIdsToNotifyAboutChatChanged, tx)
			return c.JSON(http.StatusAccepted, responseDto)
		}
	})
	if errOuter != nil {
		GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch ChatHandler) LeaveChat(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	errOuter := db.Transact(ch.db, func(tx *db.Tx) error {
		if err := tx.DeleteParticipant(userPrincipalDto.UserId, chatId); err != nil {
			return err
		}

		firstUser, err2 := tx.GetFirstParticipant(chatId)
		if err2 != nil {
			return err2
		}
		if responseDto, err := getChat(tx, ch.restClient, c, chatId, firstUser, nil); err != nil {
			return err
		} else {
			ch.notificator.NotifyAboutChangeChat(c, responseDto, responseDto.ParticipantIds, tx)
			ch.notificator.NotifyAboutDeleteChat(c, responseDto, []int64{userPrincipalDto.UserId}, tx)
			return c.JSON(http.StatusAccepted, responseDto)
		}
	})
	if errOuter != nil {
		GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
	}
	return errOuter
}

func (ch ChatHandler) CheckAccess(c echo.Context) error {
	chatId, err := GetQueryParamAsInt64(c, "chatId")
	if err != nil {
		return err
	}
	userId, err := GetQueryParamAsInt64(c, "userId")
	if err != nil {
		return err
	}
	participant, err := ch.db.IsParticipant(userId, chatId)
	if err != nil {
		return err
	}
	if participant {
		return c.NoContent(http.StatusOK)
	} else {
		return c.NoContent(http.StatusUnauthorized)
	}
}