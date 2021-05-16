package handlers

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/chat/auth"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"strings"
)

type SubscribeHandler struct {
	redis *redis.Pool
}

func NewSubscribeHandler(redis *redis.Pool) SubscribeHandler {
	return SubscribeHandler{
		redis: redis,
	}
}

type SubscribeRequestDto struct {
	Users []int64 `json:"userIds"`
}

const prefix = "subscription:user:online"

func GetUserIdFromSubscriptionIdAsString(subscriptionId string) (string) {
	split := strings.Split(subscriptionId, ":")
	return split[3]
}

func (ch SubscribeHandler) PutSubscription(c echo.Context) error {
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		GetLogEntry(c.Request()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}
	var bindTo = new(SubscribeRequestDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request()).Warnf("Error during binding to dto %v", err)
		return err
	}

	conn := ch.redis.Get()
	defer conn.Close()

	subscriptionId := fmt.Sprintf("%v:%v:%v", prefix, userPrincipalDto.UserId, userPrincipalDto.SessionId)
	if len(bindTo.Users) == 0 {
		_, err := conn.Do("DEL", subscriptionId)
		if err != nil {
			Logger.Errorf("Error during deleting user online subscription, %v", err)
			return c.NoContent(http.StatusInternalServerError)
		} else {
			return c.NoContent(http.StatusAccepted)
		}

	} else {
		args := []interface{}{subscriptionId}
		for _, x := range bindTo.Users {
			args = append(args, x)
		}
		_, err := conn.Do("SADD", args...)
		if err != nil {
			Logger.Errorf("Error during putting to user online subscription, %v", err)
			return c.NoContent(http.StatusInternalServerError)
		} else {
			return c.NoContent(http.StatusAccepted)
		}
	}
}

// e. g. get all sessions which have subscription
func (ch SubscribeHandler) GetAllUserOnlineSubscriptions() ([]string, error) {
	conn := ch.redis.Get()
	defer conn.Close()

	data, err := redis.Strings(conn.Do("KEYS", prefix + "*"))
	if err != nil {
		if err == redis.ErrNil {
			return []string{}, nil
		}
		Logger.Errorf("Error during getting user online subscriptions %v", err)
		return nil, err
	} else {
		return data, nil
	}
}

func (ch SubscribeHandler) GetUserOnlineSubscribedUsers(subscriptionId string) ([]int64, error) {
	conn := ch.redis.Get()
	defer conn.Close()

	data, err := redis.Int64s(conn.Do("SMEMBERS", subscriptionId))
	if err != nil {
		Logger.Errorf("Error during getting user online subscription content %v", err)
		return nil, err
	} else {
		return data, nil
	}
}