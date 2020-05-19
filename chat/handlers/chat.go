package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/nkonev/videochat/db"
	. "github.com/nkonev/videochat/logger"
	"github.com/nkonev/videochat/models"
)

type ChatDto struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

func GetChats(db db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		tx, err := db.Begin()
		if err != nil {
			Logger.Errorf("Error during open transaction %v", err)
			return err
		}
		chats, err := tx.GetChats(40, 0)
		if err != nil {
			Logger.Errorf("Error get chats from db %v", err)
			return err
		}
		usrs := make([]ChatDto, 0)
		for _, c := range chats {
			usrs = append(usrs, convert(c))
		}
		err = tx.Commit()
		if err != nil {
			Logger.Errorf("Error during commit transaction %v", err)
			return err
		}
		return c.JSON(200, usrs)
	}
}

func convert(c models.Chat) ChatDto {
	return ChatDto{
		Id:   c.Id,
		Name: c.Title,
	}
}
