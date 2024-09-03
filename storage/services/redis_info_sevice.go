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

const convertingOriginalPrefix = "converting:original:"
const convertingConvertedPrefix = "converting:converted:"

func (s *RedisInfoService) GetOriginalConverting(ctx context.Context, minioKeyOfOriginal string) (bool, error) {
	value, err := s.redisClient.Get(ctx, convertingOriginalPrefix+minioKeyOfOriginal).Bool()
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

func (s *RedisInfoService) SetOriginalConverting(ctx context.Context, minioKeyOfOriginal string) {
	maxConvertingDuration := viper.GetDuration("converting.maxDuration")
	s.redisClient.Set(ctx, convertingOriginalPrefix+minioKeyOfOriginal, true, maxConvertingDuration)
}

func (s *RedisInfoService) RemoveOriginalConverting(ctx context.Context, minioKeyOfOriginal string) {
	s.redisClient.Del(ctx, convertingOriginalPrefix+minioKeyOfOriginal)
}


func (s *RedisInfoService) GetConvertedConverting(ctx context.Context, minioKeyOfConverted string) (bool, error) {
	value, err := s.redisClient.Get(ctx, convertingConvertedPrefix+minioKeyOfConverted).Bool()
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

func (s *RedisInfoService) SetConvertedConverting(ctx context.Context, minioKeyOfConverted string) {
	maxConvertingDuration := viper.GetDuration("converting.maxDuration")
	s.redisClient.Set(ctx, convertingConvertedPrefix+minioKeyOfConverted, true, maxConvertingDuration)
}

func (s *RedisInfoService) RemoveConvertedConverting(ctx context.Context, minioKeyOfConverted string) {
	s.redisClient.Del(ctx, convertingConvertedPrefix+minioKeyOfConverted)
}
