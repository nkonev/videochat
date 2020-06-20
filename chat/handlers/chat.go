package handlers

import (
	"errors"
	"fmt"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/db"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

type ChatDto struct {
	Id             int64   `json:"id"`
	Name           string  `json:"name"`
	ParticipantIds []int64 `json:"participantIds"`
}

type EditChatDto struct {
	Id             int64   `json:"id"`
	Name           string  `json:"name"`
	ParticipantIds []int64 `json:"participantIds"`
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

func GetChats(db db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		page := utils.FixPageString(c.QueryParam("page"))
		size := utils.FixSizeString(c.QueryParam("size"))
		offset := utils.GetOffset(page, size)

		if chats, err := db.GetChats(userPrincipalDto.UserId, size, offset); err != nil {
			GetLogEntry(c.Request()).Errorf("Error get chats from db %v", err)
			return err
		} else {
			chatDtos := make([]*ChatDto, 0)
			for _, c := range chats {
				if ids, err := db.GetParticipantIds(c.Id); err != nil {
					return err
				} else {
					chatDtos = append(chatDtos, convertToDto(c, ids))
				}
			}
			GetLogEntry(c.Request()).Infof("Successfully returning %v chats", len(chatDtos))
			return c.JSON(200, chatDtos)
		}
	}
}

func GetChat(dbR db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		chatIdString := c.Param("id")
		chatId, err := utils.ParseInt64(chatIdString)
		if err != nil {
			return err
		}
		if chat, err := dbR.GetChat(userPrincipalDto.UserId, chatId); err != nil {
			GetLogEntry(c.Request()).Errorf("Error get chats from db %v", err)
			return err
		} else {
			if chat == nil {
				return c.NoContent(http.StatusNotFound)
			}
			ids, err := dbR.GetParticipantIds(chat.Id)
			if err != nil {
				return err
			}
			chatDto := convertToDto(chat, ids)
			GetLogEntry(c.Request()).Infof("Successfully returning %v chat", chatDto)
			return c.JSON(200, chatDto)
		}
	}
}

func convertToDto(c *db.Chat, participantIds []int64) *ChatDto {
	return &ChatDto{
		Id:             c.Id,
		Name:           c.Title,
		ParticipantIds: participantIds,
	}
}

func CreateChat(dbR db.DB) func(c echo.Context) error {
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

		result, errOuter := db.TransactWithResult(dbR, func(tx *db.Tx) (interface{}, error) {
			id, err := tx.CreateChat(convertToCreatableChat(bindTo))
			if err != nil {
				return 0, err
			}
			if err := tx.AddParticipant(userPrincipalDto.UserId, id, true); err != nil {
				return 0, err
			}
			for _, participantId := range bindTo.ParticipantIds {
				if participantId == userPrincipalDto.UserId {
					continue
				}
				if err := tx.AddParticipant(participantId, id, false); err != nil {
					return 0, err
				}
			}
			return id, err
		})
		if errOuter != nil {
			GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
			return errOuter
		} else {
			return c.JSON(http.StatusCreated, &utils.H{"id": result})
		}
	}
}

func convertToCreatableChat(d *CreateChatDto) *db.Chat {
	return &db.Chat{
		Title: d.Name,
	}
}

func DeleteChat(dbR db.DB) func(c echo.Context) error {
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
			if err := tx.DeleteChat(chatId); err != nil {
				return err
			}
			return c.JSON(http.StatusAccepted, &utils.H{"id": chatId})
		})
		if errOuter != nil {
			GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
		}
		return errOuter
	}
}

func EditChat(dbR db.DB) func(c echo.Context) error {
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

		errOuter := db.Transact(dbR, func(tx *db.Tx) error {
			if admin, err := tx.IsAdmin(userPrincipalDto.UserId, bindTo.Id); err != nil {
				return err
			} else if !admin {
				return errors.New(fmt.Sprintf("User %v is not admin of chat %v", userPrincipalDto.UserId, bindTo.Id))
			}
			if err := tx.EditChat(bindTo.Id, bindTo.Name); err != nil {
				return err
			}
			// TODO re-think about case when non-admin is trying to edit
			if err := tx.DeleteParticipantsExcept(userPrincipalDto.UserId, bindTo.Id); err != nil {
				return err
			}
			for _, participantId := range bindTo.ParticipantIds {
				if participantId == userPrincipalDto.UserId {
					continue
				}
				if err := tx.AddParticipant(participantId, bindTo.Id, false); err != nil {
					return err
				}
			}
			return c.JSON(http.StatusAccepted, &utils.H{"id": bindTo.Id})
		})
		if errOuter != nil {
			GetLogEntry(c.Request()).Errorf("Error during act transaction %v", errOuter)
		}
		return errOuter
	}
}
