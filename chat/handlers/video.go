package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

type VideoHandler struct {
	db         db.DB
	restClient client.RestClient
}

func NewVideoHandler(db db.DB, restClient client.RestClient) VideoHandler {
	return VideoHandler{db, restClient}
}

func (vh VideoHandler) GetOpenviduToken(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	if err != nil {
		return err
	}

	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	isParticipant, err := vh.db.IsParticipant(userPrincipalDto.UserId, chatId)
	if err != nil {
		return err
	}
	if !isParticipant {
		return c.JSON(http.StatusUnauthorized, &utils.H{"message": "You have no acces to this chat"})
	}

	session, err := vh.restClient.CreateOpenviduSession(chatId)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Unable to create openvidu session for chat %v %v", chatId, err)
		return err
	}
	token, err := vh.restClient.CreateOpenviduConnection(session)
	if err != nil {
		GetLogEntry(c.Request()).Errorf("Unable to create openvidu connection for chat %v %v", chatId, err)
		return err
	}
	return c.JSON(http.StatusOK, &utils.H{"token": token})
}
