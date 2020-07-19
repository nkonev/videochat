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
)

type ChatDto = dto.ChatDto

type ChatWrapper struct {
	Data  []*ChatDto `json:"data"`
	Count int64      `json:"totalCount"` // total chat number for this user
}

type EditChatDto struct {
	Id int64 `json:"id"`
	CreateChatDto
}

type CreateChatDto struct {
	Name           string  `json:"name"`
	ParticipantIds []int64 `json:"participantIds"`
}

func (a *CreateChatDto) Validate() error {
	return validation.ValidateStruct(a, validation.Field(&a.Name, validation.Required, validation.Length(1, 256)))
}

func (a *EditChatDto) Validate() error {
	return validation.ValidateStruct(a, validation.Field(&a.Name, validation.Required, validation.Length(1, 256)), validation.Field(&a.Id, validation.Required))
}

func GetChats(db db.DB, restClient client.RestClient) func(c echo.Context) error {
	return func(c echo.Context) error {
		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		page := utils.FixPageString(c.QueryParam("page"))
		size := utils.FixSizeString(c.QueryParam("size"))
		offset := utils.GetOffset(page, size)
		searchString := c.QueryParam("searchString")

		if dbChats, err := db.GetChatsWithParticipants(userPrincipalDto.UserId, size, offset, searchString, userPrincipalDto); err != nil {
			GetLogEntry(c.Request()).Errorf("Error get chats from db %v", err)
			return err
		} else {
			chatDtos := make([]*ChatDto, 0)
			for _, cc := range dbChats {
				cd := convertToDto(cc, []*dto.User{})
				chatDtos = append(chatDtos, cd)
			}

			var participantIdSet = map[int64]bool{}
			for _, chatDto := range chatDtos {
				for _, participantId := range chatDto.ParticipantIds {
					participantIdSet[participantId] = true
				}
			}
			var users = getUsersRemotelyOrEmpty(participantIdSet, restClient, c)
			for _, chatDto := range chatDtos {
				for _, participantId := range chatDto.ParticipantIds {
					user := users[participantId]
					if user != nil {
						chatDto.Participants = append(chatDto.Participants, user)
					}
				}
			}

			userChatCount, err := db.CountChatsPerUser(userPrincipalDto.UserId)
			if err != nil {
				return errors.New("Error during getting user chat count")
			}
			GetLogEntry(c.Request()).Infof("Successfully returning %v chats", len(chatDtos))
			return c.JSON(200, ChatWrapper{Data: chatDtos, Count: userChatCount})
		}
	}
}

func getChat(dbR db.CommonOperations, restClient client.RestClient, c echo.Context, chatId int64, userPrincipalDto *auth.AuthResult) (*ChatDto, error) {
	if cc, err := dbR.GetChatWithParticipants(userPrincipalDto.UserId, chatId, userPrincipalDto); err != nil {
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
		chatDto := convertToDto(cc, users)

		return chatDto, nil
	}
}

func GetChat(dbR db.DB, restClient client.RestClient) func(c echo.Context) error {
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

		if chat, err := getChat(&dbR, restClient, c, chatId, userPrincipalDto); err != nil {
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
}

func convertToDto(c *db.ChatWithParticipants, users []*dto.User) *ChatDto {
	return &ChatDto{
		Id:                 c.Id,
		Name:               c.Title,
		ParticipantIds:     c.ParticipantsIds,
		Participants:       users,
		CanEdit:            null.BoolFrom(c.IsAdmin),
		LastUpdateDateTime: c.LastUpdateDateTime,
	}
}

func CreateChat(dbR db.DB, notificator notifications.Notifications, restClient client.RestClient) func(c echo.Context) error {
	return func(c echo.Context) error {
		var bindTo = new(CreateChatDto)
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

		errOuter := db.Transact(dbR, func(tx *db.Tx) error {
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

			responseDto, err := getChat(tx, restClient, c, id, userPrincipalDto)
			if err != nil {
				return err
			}

			notificator.NotifyAboutNewChat(c, responseDto, responseDto.ParticipantIds, tx)
			return c.JSON(http.StatusCreated, responseDto)
		})
		if errOuter != nil {
			GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
		}
		return errOuter
	}
}

func convertToCreatableChat(d *CreateChatDto) *db.Chat {
	return &db.Chat{
		Title: d.Name,
	}
}

func DeleteChat(dbR db.DB, notificator notifications.Notifications) func(c echo.Context) error {
	return func(c echo.Context) error {
		chatId, err := GetPathParamAsInt64(c, "id")
		if err != nil {
			return err
		}

		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		errOuter := db.Transact(dbR, func(tx *db.Tx) error {
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
			cd := &ChatDto{
				Id: chatId,
			}
			notificator.NotifyAboutDeleteChat(c, cd, ids, tx)
			return c.JSON(http.StatusAccepted, &utils.H{"id": chatId})
		})
		if errOuter != nil {
			GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
		}
		return errOuter
	}
}

func getIndexOf(ids []int64, elem int64) int {
	for i := 0; i < len(ids); i++ {
		if ids[i] == elem {
			return i
		}
	}
	return -1
}

func contains(ids []int64, elem int64) bool {
	return getIndexOf(ids, elem) != -1
}

func EditChat(dbR db.DB, notificator notifications.Notifications, restClient client.RestClient) func(c echo.Context) error {
	return func(c echo.Context) error {
		var bindTo = new(EditChatDto)
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

		var userIdsToNotifyAboutChatCreated []int64
		var userIdsToNotifyAboutChatDeleted []int64
		var userIdsToNotifyAboutChatChanged []int64
		errOuter := db.Transact(dbR, func(tx *db.Tx) error {
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
				if !contains(existsChatParticipantIdsFromDatabase, participantIdFromRequest) {
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
				if !contains(bindTo.ParticipantIds, participantIdFromDatabase) && participantIdFromDatabase != userPrincipalDto.UserId {
					if err := tx.DeleteParticipant(participantIdFromDatabase, bindTo.Id); err != nil {
						return err
					}
					userIdsToNotifyAboutChatDeleted = append(userIdsToNotifyAboutChatDeleted, participantIdFromDatabase)
				}
			}

			if responseDto, err := getChat(tx, restClient, c, bindTo.Id, userPrincipalDto); err != nil {
				return err
			} else {
				notificator.NotifyAboutNewChat(c, responseDto, userIdsToNotifyAboutChatCreated, tx)
				notificator.NotifyAboutDeleteChat(c, responseDto, userIdsToNotifyAboutChatDeleted, tx)
				notificator.NotifyAboutChangeChat(c, responseDto, userIdsToNotifyAboutChatChanged, tx)
				return c.JSON(http.StatusAccepted, responseDto)
			}
		})
		if errOuter != nil {
			GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
		}
		return errOuter
	}
}
