package handlers

import (
	"errors"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/db"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"time"
)

type NewMessageDto struct {
	Text   string `json:"text"`
	ChatId int64  `json:"chatId"`
}

type EditMessageDto struct {
	Id     int64  `json:"id"`
	Text   string `json:"text"`
	ChatId int64  `json:"chatId"`
}

type DisplayMessageDto struct {
	Id             int64     `json:"id"`
	Text           string    `json:"text"`
	ChatId         int64     `json:"chatId"`
	OwnerId        int64     `json:"ownerId"`
	CreateDateTime time.Time `json:"createDateTime"`
	EditDateTime   null.Time `json:"editDateTime"`
}

func GetMessages(dbR db.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
		if !ok {
			GetLogEntry(c.Request()).Errorf("Error during getting auth context")
			return errors.New("Error during getting auth context")
		}

		page := utils.FixPageString(c.QueryParam("page"))
		size := utils.FixSizeString(c.QueryParam("size"))
		offset := utils.GetOffset(page, size)

		chatIdString := c.Param("id")
		chatId, err := utils.ParseInt64(chatIdString)
		if err != nil {
			return err
		}

		if messages, err := dbR.GetMessages(chatId, userPrincipalDto.UserId, size, offset); err != nil {
			GetLogEntry(c.Request()).Errorf("Error get messages from db %v", err)
			return err
		} else {
			messageDtos := make([]*DisplayMessageDto, 0)
			for _, c := range messages {
				messageDtos = append(messageDtos, convertToMessageDto(c))
			}
			GetLogEntry(c.Request()).Infof("Successfully returning %v messages", len(messageDtos))
			return c.JSON(200, messageDtos)
		}
	}
}

func convertToMessageDto(dbMessage *db.Message) *DisplayMessageDto {
	return &DisplayMessageDto{
		Id:             dbMessage.Id,
		Text:           dbMessage.Text,
		ChatId:         dbMessage.ChatId,
		OwnerId:        dbMessage.OwnerId,
		CreateDateTime: dbMessage.CreateDateTime,
		EditDateTime:   dbMessage.EditDateTime,
	}
}
