package services

import (
	"context"
	"fmt"
	redisV8 "github.com/go-redis/redis/v8"
	"nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/utils"
)

type DialRedisService struct {
	rabbitMqPublisher *producer.RabbitInvitePublisher
	redisClient       *redisV8.Client
}

func NewDialRedisService(rabbitMqPublisher *producer.RabbitInvitePublisher, redisClient *redisV8.Client) *DialRedisService {
	return &DialRedisService{
		rabbitMqPublisher: rabbitMqPublisher,
		redisClient:       redisClient,
	}
}

func (s *DialRedisService) AddToDialList(ctx context.Context, userId, chatId int64, behalfUserId int64, behalfLogin string) error {
	add := s.redisClient.SAdd(ctx, fmt.Sprintf("dialchat%v", chatId), userId)
	err := add.Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial %v", err)
		return err
	}
	err = s.redisClient.HSet(ctx, fmt.Sprintf("dialmeta%v", chatId), "behalfUserId", behalfUserId, "behalfLogin", behalfLogin).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during setting dial metadata %v", err)
		return err
	}
	return nil
}

func (s *DialRedisService) GerDialMetadata(ctx context.Context, chatId int64) (int64, string, error) {

	all := s.redisClient.HGetAll(ctx, fmt.Sprintf("dialmeta%v", chatId))
	if all.Err() != nil {
		logger.GetLogEntry(ctx).Errorf("Error during getting dial metadata %v", all.Err())
		return 0, "", all.Err()
	}
	val := all.Val()
	s2 := val["behalfUserId"]
	parsedBehalfUserId, err := utils.ParseInt64(s2)
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during parsing userId %v", err)
		return 0, "", err
	}
	return parsedBehalfUserId, val["behalfLogin"], nil
}

func (s *DialRedisService) RemoveFromDialList(ctx context.Context, userId, chatId int64) error {
	add := s.redisClient.SRem(ctx, fmt.Sprintf("dialchat%v", chatId), userId)
	// TODO remove "dialchat%v" on zero members
	// TODO "dialmeta%v" on zero members
	return add.Err()
}

func (s *DialRedisService) GetDialChats(ctx context.Context) ([]int64, error) {
	ret, err := s.redisClient.Keys(ctx, "dialchat*").Result()
	if err == redisV8.Nil {
		return []int64{}, nil
	}
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during scanning chats %v", err)
		return nil, err
	}

	var ret0 = []int64{}
	for _, vl := range ret {
		var str string
		_, err := fmt.Sscanf(vl, "dialchat%s", &str)
		if err != nil {
			logger.GetLogEntry(ctx).Warnf("Error during parsing userId %v", err)
			continue
		}
		parseInt64, err := utils.ParseInt64(str)
		if err != nil {
			logger.GetLogEntry(ctx).Warnf("Error during parsing userId %v", err)
			continue
		}
		ret0 = append(ret0, parseInt64)
	}
	return ret0, nil
}

func (s *DialRedisService) GetUsersToDial(ctx context.Context, chatId int64) ([]int64, error) {
	members, err := s.redisClient.SMembers(ctx, fmt.Sprintf("dialchat%v", chatId)).Result()
	if err == redisV8.Nil {
		return []int64{}, nil
	}
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during scanning users of chat %v", err)
		return nil, err
	}
	var ret = []int64{}
	for _, vl := range members {
		parseInt64, err := utils.ParseInt64(vl)
		if err != nil {
			logger.GetLogEntry(ctx).Warnf("Error during parsing userId %v", err)
			continue
		}
		ret = append(ret, parseInt64)
	}
	return ret, nil
}
