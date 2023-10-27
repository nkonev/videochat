package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
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
	Codec   *string     `json:"codec"`
}

func (h *ConfigHandler) GetConfig(c echo.Context) error {

	frontendConfig := h.config.FrontendConfig
	var responseSliceFrontendConfig = FrontendConfigDto{}

	responseSliceFrontendConfig.VideoResolution = frontendConfig.VideoResolution
	responseSliceFrontendConfig.ScreenResolution = frontendConfig.ScreenResolution
	responseSliceFrontendConfig.VideoSimulcast = frontendConfig.VideoSimulcast
	responseSliceFrontendConfig.ScreenSimulcast = frontendConfig.ScreenSimulcast
	responseSliceFrontendConfig.RoomDynacast = frontendConfig.RoomDynacast
	responseSliceFrontendConfig.RoomAdaptiveStream = frontendConfig.RoomAdaptiveStream
	responseSliceFrontendConfig.Codec = frontendConfig.Codec

	return c.JSON(http.StatusOK, responseSliceFrontendConfig)
}
