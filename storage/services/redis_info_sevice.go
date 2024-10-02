package services

import (
	"context"
	redisV9 "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type RedisInfoService struct {
	redisClient *redisV9.Client
}

func NewRedisInfoService(redisClient *redisV9.Client) *RedisInfoService {
	return &RedisInfoService{
		redisClient: redisClient,
	}
}

const convertingOriginalPrefix = "converting:original:"
const convertingConvertedPrefix = "converting:converted:"

func (s *RedisInfoService) GetOriginalConverting(ctx context.Context, minioKeyOfOriginal string) (bool, error) {
	value, err := s.redisClient.Get(ctx, convertingOriginalPrefix+minioKeyOfOriginal).Bool()
	if err != nil {
		if err == redisV9.Nil {
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
		if err == redisV9.Nil {
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
