package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/chat/client"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

type VideoHandler struct {
	restClient client.RestClient
}

func NewVideoHandler(restClient client.RestClient) VideoHandler {
	return VideoHandler{restClient}
}

func (vh VideoHandler) GetConfiguration(c echo.Context) error {
	slice := viper.GetStringSlice("iceServers")
	return c.JSON(200, slice)
}

func (vh VideoHandler) GetOpenviduToken(c echo.Context) error {
	chatId, err := GetPathParamAsInt64(c, "id")
	// todo check access
	if err != nil {
		return err
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
