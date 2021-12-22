package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/redis"
	"nkonev.name/chat/utils"
	"strings"
)

type UserOnlineHandler struct {
	onlineStorage redis.OnlineStorage
}

func NewOnlineHandler(onlineStorage redis.OnlineStorage) *UserOnlineHandler {
	return &UserOnlineHandler{
		onlineStorage: onlineStorage,
	}
}

func (h *UserOnlineHandler) GetOnlineUsers(context echo.Context) error {
	param := context.QueryParam("participantIds")
	Logger.Printf("See here - %v", param)
	split := strings.Split(param, ",")

	var arr []UserOnlineChanged = make([]UserOnlineChanged, 0)

	for _, participantId := range split {
		if participantId == "" {
			continue
		}
		userIdInt64, err := utils.ParseInt64(participantId)
		if err != nil {
			Logger.Warnf("Error during parsing participantId %v %v %v", param, userIdInt64, err)
			continue
		}
		online, err := h.onlineStorage.GetUserOnline(userIdInt64)
		if err != nil {
			Logger.Warnf("Error during getting online from participantId %v %v", userIdInt64, err)
			continue
		}
		arr = append(arr, UserOnlineChanged{
			UserId: userIdInt64,
			Online: online != 0,
		})

	}

	return context.JSON(http.StatusOK, arr)
}
