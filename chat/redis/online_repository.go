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

func (recv OnlineStorage) PutUserOnline(userId int64, expirationTimestamp int64) {
	conn := recv.redis.Get()
	defer conn.Close()

	if _, err := conn.Do("INCR", fmt.Sprintf("online:%v", userId)); err != nil {
		logger.Logger.Errorf("Error during setting online for user %v", userId)
		return
	}
	if _, err := conn.Do("EXPIRE", fmt.Sprintf("online:%v", userId), expirationTimestamp); err != nil {
		logger.Logger.Errorf("Error during setting online for user %v", userId)
		return
	}
}

func (recv OnlineStorage) RemoveUserOnline(userId int64) {
	conn := recv.redis.Get()
	defer conn.Close()

	if data, err := redis.Int64(conn.Do("DECR", fmt.Sprintf("online:%v", userId))); err != nil {
		logger.Logger.Errorf("Error during decrementing online for user %v", userId)
		return
	} else {
		if data == 0 {
			if _, err := conn.Do("DEL", fmt.Sprintf("online:%v", userId)); err != nil {
				logger.Logger.Errorf("Error during removing online for user %v", userId)
				return
			}
		}
	}
}

func (recv OnlineStorage) GetUserOnline(userId int64) (int64, error) {
	conn := recv.redis.Get()
	defer conn.Close()

	data, err := redis.Int64(conn.Do("GET", fmt.Sprintf("online:%v", userId)))
	if err != nil {
		if err == redis.ErrNil {
			return 0, nil
		}
		logger.Logger.Errorf("Error during getting online for user %v", userId)
		return 0, err
	} else {
		return data, nil
	}
}
