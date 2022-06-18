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
	RtcConfig      *RtcConfig `json:"rtcConfiguration"`
	PreferredCodec string     `json:"codec"`
	Resolution     string     `json:"resolution"`
}

func (h *ConfigHandler) GetConfig(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	chatId, err := utils.ParseInt64(c.Param("chatId"))
	if err != nil {
		return err
	}
	if ok, err := h.chatClient.CheckAccess(userPrincipalDto.UserId, chatId); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}

	frontendConfig := h.config.FrontendConfig
	var responseSliceFrontendConfig = FrontendConfigDto{}

	for _, s := range frontendConfig.ICEServers {
		if responseSliceFrontendConfig.RtcConfig == nil {
			responseSliceFrontendConfig.RtcConfig = &RtcConfig{}
		}
		var newElement = ICEServerConfigDto{
			URLs:       s.ICEServerConfig.URLs,
			Username:   s.ICEServerConfig.Username,
			Credential: s.ICEServerConfig.Credential,
		}
		responseSliceFrontendConfig.RtcConfig.ICEServers = append(responseSliceFrontendConfig.RtcConfig.ICEServers, newElement)
	}
	responseSliceFrontendConfig.PreferredCodec = frontendConfig.PreferredCodec
	responseSliceFrontendConfig.Resolution = frontendConfig.Resolution

	return c.JSON(http.StatusOK, responseSliceFrontendConfig)
}
