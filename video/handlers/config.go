package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/utils"
)

type ConfigHandler struct {
	chatClient *client.RestClient
	config     *config.ExtendedConfig
}

func NewConfigHandler(chatClient *client.RestClient, config *config.ExtendedConfig) *ConfigHandler {
	return &ConfigHandler{chatClient: chatClient, config: config}
}

type RtcConfig struct {
	ICEServers []ICEServerConfigDto `json:"iceServers"`
}

type ICEServerConfigDto struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username"`
	Credential string   `json:"credential"`
}

type FrontendConfigDto struct {
	RtcConfig          *RtcConfig `json:"rtcConfiguration"`
	VideoResolution    string     `json:"videoResolution"`
	ScreenResolution   string     `json:"screenResolution"`
	VideoSimulcast     *bool      `json:"videoSimulcast"`
	ScreenSimulcast    *bool      `json:"screenSimulcast"`
	RoomDynacast       *bool      `json:"roomDynacast"`
	RoomAdaptiveStream *bool      `json:"roomAdaptiveStream"`
}

func (h *ConfigHandler) GetConfig(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.chatClient.CheckAccess(userPrincipalDto.UserId, chatId, c.Request().Context()); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	frontendConfig := h.config.FrontendConfig
	var responseSliceFrontendConfig = FrontendConfigDto{}

	responseSliceFrontendConfig.VideoResolution = frontendConfig.VideoResolution
	responseSliceFrontendConfig.ScreenResolution = frontendConfig.ScreenResolution
	responseSliceFrontendConfig.VideoSimulcast = frontendConfig.VideoSimulcast
	responseSliceFrontendConfig.ScreenSimulcast = frontendConfig.ScreenSimulcast
	responseSliceFrontendConfig.RoomDynacast = frontendConfig.RoomDynacast
	responseSliceFrontendConfig.RoomAdaptiveStream = frontendConfig.RoomAdaptiveStream

	return c.JSON(http.StatusOK, responseSliceFrontendConfig)
}
