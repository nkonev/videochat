package services

import (
	"context"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

type RedisInfoService struct {
	redisClient *redisV8.Client
}

func NewRedisInfoService(redisClient *redisV8.Client) *RedisInfoService {
	return &RedisInfoService{
		redisClient: redisClient,
	}
}

const convertingPrefix = "converting:"

func (s *RedisInfoService) GetConverting(ctx context.Context, minioKey string) (bool, error) {
	value, err := s.redisClient.Get(ctx, convertingPrefix+minioKey).Bool()
	if err != nil {
		if err == redisV8.Nil {
			return false, nil
		} else {
			return false, err
		}
	} else {
		return value, nil
	}
}

func (s *RedisInfoService) SetConverting(ctx context.Context, minioKey string) {
	maxConvertingDuration := viper.GetDuration("converting.maxDuration")
	s.redisClient.Set(ctx, convertingPrefix+minioKey, true, maxConvertingDuration)
}

func (s *RedisInfoService) RemoveConverting(ctx context.Context, minioKey string) {
	s.redisClient.Del(ctx, convertingPrefix+minioKey)
}
