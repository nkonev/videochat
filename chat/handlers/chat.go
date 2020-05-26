package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/db"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

type ChatDto struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

type CreateChatDto struct {
	Name string `json:"name"`
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
				chatDtos = append(chatDtos, convertToDto(c))
			}
			GetLogEntry(c.Request()).Infof("Successfully returning %v chats", len(chatDtos))
			return c.JSON(200, chatDtos)
		}
	}
}

func GetChat(db db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		chatIdString := c.Param("id")
		i, err := utils.ParseInt64(chatIdString)
		if err != nil {
			return err
		}

		if chat, err := db.GetChat(userPrincipalDto.UserId, i); err != nil {
			GetLogEntry(c.Request()).Errorf("Error get chats from db %v", err)
			return err
		} else {
			chatDto := convertToDto(chat)
			GetLogEntry(c.Request()).Infof("Successfully returning %v chat", chatDto)
			return c.JSON(200, chatDto)
		}
	}
}

func convertToDto(c *db.Chat) *ChatDto {
	return &ChatDto{
		Id:   c.Id,
		Name: c.Title,
	}
}

func CreateChat(db db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		var bindTo = new(CreateChatDto)
		if err := c.Bind(bindTo); err != nil {
			GetLogEntry(c.Request()).Errorf("Error during binding to dto %v", err)
			return err
		}

		if tx, err := db.Begin(); err != nil {
			GetLogEntry(c.Request()).Errorf("Error during open transaction %v", err)
			return err
		} else {
			var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
			if !ok {
				GetLogEntry(c.Request()).Errorf("Error during getting auth context")
				tx.SafeRollback()
				return errors.New("Error during getting auth context")
			}

			if id, err := tx.CreateChat(convertToCreatableChat(bindTo, userPrincipalDto)); err != nil {
				GetLogEntry(c.Request()).Errorf("Error get chats from db %v", err)
				tx.SafeRollback()
				return err
			} else {
				if err := tx.Commit(); err != nil {
					GetLogEntry(c.Request()).Errorf("Error during commit transaction %v", err)
					return err
				}
				GetLogEntry(c.Request()).Infof("Successfully created chat %v", bindTo)
				return c.JSON(http.StatusCreated, &utils.H{"id": id})
			}
		}
	}
}

func convertToCreatableChat(d *CreateChatDto, a *auth.AuthResult) *db.Chat {
	return &db.Chat{
		Title:   d.Name,
		OwnerId: a.UserId,
	}
}

func DeleteChat(db db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		chatIdString := c.Param("id")
		i, err := utils.ParseInt64(chatIdString)
		if err != nil {
			return err
		}
		return db.DeleteChat(i)
	}
}

func EditChat(db db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		var bindTo = new(ChatDto)
		if err := c.Bind(bindTo); err != nil {
			GetLogEntry(c.Request()).Errorf("Error during binding to dto %v", err)
			return err
		}

		if tx, err := db.Begin(); err != nil {
			GetLogEntry(c.Request()).Errorf("Error during open transaction %v", err)
			return err
		} else {
			var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
			if !ok {
				GetLogEntry(c.Request()).Errorf("Error during getting auth context")
				tx.SafeRollback()
				return errors.New("Error during getting auth context")
			}

			if err := tx.EditChat(bindTo.Id, userPrincipalDto.UserId, bindTo.Name); err != nil {
				GetLogEntry(c.Request()).Errorf("Error get chats from db %v", err)
				tx.SafeRollback()
				return err
			} else {
				if err := tx.Commit(); err != nil {
					GetLogEntry(c.Request()).Errorf("Error during commit transaction %v", err)
					return err
				}
				GetLogEntry(c.Request()).Infof("Successfully updated chat %v", bindTo)
				return c.NoContent(http.StatusOK)
			}
		}
	}
}
