package redis

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"nkonev.name/chat/logger"
)

type OnlineStorage struct {
	redis *redis.Pool
}

func NewOnlineStorage(redis *redis.Pool) OnlineStorage {
	return OnlineStorage{
		redis: redis,
	}
}

func (recv OnlineStorage) PutUserOnline(userId int64) {
	conn := recv.redis.Get()
	defer conn.Close()

	if _, err := conn.Do("SET", fmt.Sprintf("online:%v", userId), "true"); err != nil {
		logger.Logger.Errorf("Error during setting online for user %v", userId)
		return
	}
}

func (recv OnlineStorage) RemoveUserOnline(userId int64) {
	conn := recv.redis.Get()
	defer conn.Close()

	if _, err := conn.Do("DEL", fmt.Sprintf("online:%v", userId)); err != nil {
		logger.Logger.Errorf("Error during removing online for user %v", userId)
		return
	}
}

func (recv OnlineStorage) GetUserOnline(userId int64) (bool, error) {
	conn := recv.redis.Get()
	defer conn.Close()

	data, err := redis.Bool(conn.Do("GET", fmt.Sprintf("online:%v", userId)))
	if err != nil {
		if err == redis.ErrNil {
			return false, nil
		}
		logger.Logger.Errorf("Error during getting online for user %v", userId)
		return false, err
	} else {
		return data, nil
	}
}