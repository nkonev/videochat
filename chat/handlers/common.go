package handlers

import (
	"github.com/labstack/echo/v4"
	"nkonev.name/chat/client"
	"nkonev.name/chat/handlers/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

func getOwners(owners map[int64]bool, restClient client.RestClient, c echo.Context) (map[int64]*dto.User, error) {
	var ownerIds = utils.SetToArray(owners)
	length := len(ownerIds)
	Logger.Infof("Requested user length is %v", length)
	if length == 0 {
		return map[int64]*dto.User{}, nil
	}
	users, err := restClient.GetUsers(ownerIds, c.Request().Context())
	if err != nil {
		return nil, err
	}
	var ownersObjects = map[int64]*dto.User{}
	for _, u := range users {
		ownersObjects[u.Id] = u
	}
	return ownersObjects, nil
}
