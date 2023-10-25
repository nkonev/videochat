package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	lkauth "github.com/livekit/protocol/auth"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/utils"
)

type TokenHandler struct {
	chatClient *client.RestClient
	config     *config.ExtendedConfig
}

type TokenResponse struct {
	Token string `json:"token"`
}

func NewTokenHandler(chatClient *client.RestClient, cfg *config.ExtendedConfig) *TokenHandler {
	return &TokenHandler{chatClient: chatClient, config: cfg}
}

func (h *TokenHandler) GetTokenHandler(c echo.Context) error {
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

	// https://docs.livekit.io/guides/getting-started/#generating-access-tokens-(jwt)
	// https://github.com/nkonev/videochat/blob/8fd81bccbe5f552de1ca123e2ba855dfe814cf66/development.md#generate-livekit-token

	aKey := h.config.LivekitConfig.Api.Key
	aSecret := h.config.LivekitConfig.Api.Secret
	aRoomId := utils.GetRoomNameFromId(chatId)

	token, err := h.getJoinToken(aKey, aSecret, aRoomId, userPrincipalDto)
	if err != nil {
		Logger.Errorf("Error during getting token, userId=%v, chatId=%v, error=%v", userPrincipalDto.UserId, chatId, err)
		return err
	}
	return c.JSON(http.StatusOK, TokenResponse{
		Token: token,
	})
}

func (h *TokenHandler) getJoinToken(apiKey, apiSecret, room string, authResult *auth.AuthResult) (string, error) {
	canPublish := true
	canSubscribe := true

	aId := utils.MakeIdentityFromUserId(authResult.UserId)

	at := lkauth.NewAccessToken(apiKey, apiSecret)
	grant := &lkauth.VideoGrant{
		RoomJoin:     true,
		Room:         room,
		CanPublish:   &canPublish,
		CanSubscribe: &canSubscribe,
	}

	mds, err := utils.MakeMetadata(authResult.UserId, authResult.UserLogin, authResult.Avatar)
	if err != nil {
		return "", err
	}

	validFor := viper.GetDuration("videoTokenValidTime")
	at.AddGrant(grant).
		SetIdentity(aId).SetValidFor(validFor).SetMetadata(mds)

	return at.ToJWT()
}
