package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	lkauth "github.com/livekit/protocol/auth"
	"net/http"
	"nkonev.name/video/auth"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/utils"
	"time"
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

type MetadataDto struct {
	UserId int64  `json:"userId"`
	Login  string `json:"login"`
	Avatar string `json:"avatar"` // url
}

func (h *TokenHandler) getJoinToken(apiKey, apiSecret, room string, authResult *auth.AuthResult) (string, error) {
	canPublish := true
	canSubscribe := true

	aId := fmt.Sprintf("%v_%v", authResult.UserId, uuid.New().String())

	at := lkauth.NewAccessToken(apiKey, apiSecret)
	grant := &lkauth.VideoGrant{
		RoomJoin:     true,
		Room:         room,
		CanPublish:   &canPublish,
		CanSubscribe: &canSubscribe,
	}
	md := &MetadataDto{
		UserId: authResult.UserId,
		Login:  authResult.UserLogin,
		Avatar: authResult.Avatar,
	}

	bytes, err := json.Marshal(md)
	if err != nil {
		return "", err
	}

	mds := string(bytes)

	at.AddGrant(grant).
		SetIdentity(aId).SetValidFor(time.Hour).SetMetadata(mds)

	return at.ToJWT()
}
