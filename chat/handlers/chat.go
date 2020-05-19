package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/nkonev/videochat/auth"
	"github.com/nkonev/videochat/db"
	. "github.com/nkonev/videochat/logger"
	"github.com/nkonev/videochat/utils"
)

type ChatDto struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

func GetChats(db db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		if tx, err := db.Begin(); err != nil {
			GetLogEntry(c.Request()).Errorf("Error during open transaction %v", err)
			return err
		} else {
			var userPrincipalDto = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)

			if chats, err := tx.GetChats(userPrincipalDto.UserId, 40, 0); err != nil {
				GetLogEntry(c.Request()).Errorf("Error get chats from db %v", err)
				tx.SafeRollback()
				return err
			} else {
				chatDtos := make([]ChatDto, 0)
				for _, c := range chats {
					chatDtos = append(chatDtos, convert(c))
				}
				if err := tx.Commit(); err != nil {
					GetLogEntry(c.Request()).Errorf("Error during commit transaction %v", err)
					return err
				}
				GetLogEntry(c.Request()).Infof("Successfully returning %v chats", len(chatDtos))
				return c.JSON(200, chatDtos)
			}
		}
	}
}

func convert(c db.Chat) ChatDto {
	return ChatDto{
		Id:   c.Id,
		Name: c.Title,
	}
}
