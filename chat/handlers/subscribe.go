package handlers

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
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

func GetUserIdFromSubscriptionId(subscriptionId string) (int64, error) {
	split := strings.Split(subscriptionId, ":")
	return utils.ParseInt64(split[3])
}

type SubscriptionResponse struct {
	Ttl int64 `json:"ttl"` // in seconds
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

	subscriptionId := buildSubscriptionId(userPrincipalDto.UserId, userPrincipalDto.SessionId)
	if len(bindTo.Users) == 0 {
		_, err := conn.Do("DEL", subscriptionId)
		if err != nil {
			Logger.Errorf("Error during deleting user online subscription, %v", err)
			return c.NoContent(http.StatusInternalServerError)
		} else {
			return c.NoContent(http.StatusAccepted)
		}

	} else {
		ttl := viper.GetInt64("subscription.user.online.ttl") // seconds
		if ttl == 0 {
			ttl = 30
		}

		args := []interface{}{subscriptionId}
		for _, x := range bindTo.Users {
			args = append(args, x)
		}
		_, err := conn.Do("SADD", args...)
		if err != nil {
			Logger.Errorf("Error during putting to user online subscription, %v", err)
			return c.NoContent(http.StatusInternalServerError)
		} else {
			_, err := conn.Do("EXPIRE", subscriptionId, ttl)
			if err != nil {
				Logger.Errorf("Error during setting expiration, %v", err)
				return c.NoContent(http.StatusInternalServerError)
			}
			return c.JSON(http.StatusAccepted, SubscriptionResponse{Ttl: ttl})
		}
	}
}

func buildSubscriptionId(userId int64, sessionId string) string {
	subscriptionId := fmt.Sprintf("%v:%v:%v", prefix, userId, sessionId)
	return subscriptionId
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