package services

import (
	"context"
	"fmt"
	redisV8 "github.com/go-redis/redis/v8"
	"nkonev.name/video/logger"
	"nkonev.name/video/utils"
)

type DialRedisRepository struct {
	redisClient *redisV8.Client
}

func NewDialRedisRepository(redisClient *redisV8.Client) *DialRedisRepository {
	return &DialRedisRepository{
		redisClient: redisClient,
	}
}

const behalfUserIdConstant = "behalfUserId" // also it is dial owner - person who starts the dial

const NoUser = -1

func (s *DialRedisRepository) AddToDialList(ctx context.Context, userId, chatId int64, behalfUserId int64) error {
	add := s.redisClient.SAdd(ctx, dialChatMembersKey(chatId), userId)
	err := add.Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial %v", err)
		return err
	}
	err = s.redisClient.HSet(ctx, dialMetaKey(chatId), behalfUserIdConstant, behalfUserId).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during setting dial metadata %v", err)
		return err
	}
	return nil
}

func dialChatMembersKey(chatId int64) string {
	return fmt.Sprintf("dialchat%v", chatId)
}

func getAllDialChats() string {
	return "dialchat*"
}

func dialMetaKey(chatId int64) string {
	return fmt.Sprintf("dialmeta%v", chatId)
}

func chatIdFromKey(key string) (int64, error) {
	var str string
	_, err := fmt.Sscanf(key, "dialchat%s", &str)
	if err != nil {
		return 0, err
	}
	parseInt64, err := utils.ParseInt64(str)
	if err != nil {
		return 0, err
	}
	return parseInt64, nil
}

func (s *DialRedisRepository) GetDialMetadata(ctx context.Context, chatId int64) (int64, error) {
	val, err := s.redisClient.HGetAll(ctx, dialMetaKey(chatId)).Result()
	//if err == redisV8.Nil {
	//	return -1, "", nil
	//}
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during getting dial metadata %v", err)
		return 0, err
	}
	if len(val) == 0 {
		return NoUser, nil
	}

	s2 := val[behalfUserIdConstant]
	parsedBehalfUserId, err := utils.ParseInt64(s2)
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during parsing userId %v", err)
		return 0, err
	}
	return parsedBehalfUserId, nil
}

func (s *DialRedisRepository) RemoveFromDialList(ctx context.Context, userId, chatId int64) error {
	_, err := s.redisClient.SRem(ctx, dialChatMembersKey(chatId), userId).Result()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during performing SREM %v", err)
		return err
	}

	card, err := s.redisClient.SCard(ctx, dialChatMembersKey(chatId)).Result()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during performing SCARD %v", err)
		return err
	}
	if card == 0 {
		// remove "dialchat" on zero members
		err = s.redisClient.Del(ctx, dialChatMembersKey(chatId)).Err()
		if err != nil {
			logger.GetLogEntry(ctx).Errorf("Error during deleting ChatMembers %v", err)
			return err
		}

		// remove "dialmeta" on zero members
		err = s.redisClient.Del(ctx, dialMetaKey(chatId)).Err()
		if err != nil {
			logger.GetLogEntry(ctx).Errorf("Error during deleting dialMeta %v", err)
			return err
		}
	}
	return nil
}

func (s *DialRedisRepository) RemoveDial(ctx context.Context, chatId int64) error {
	// remove "dialchat"
	err := s.redisClient.Del(ctx, dialChatMembersKey(chatId)).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during performing SREM %v", err)
		return err
	}

	// remove "dialmeta"
	err = s.redisClient.Del(ctx, dialMetaKey(chatId)).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during deleting dialMeta %v", err)
		return err
	}
	return nil
}

func (s *DialRedisRepository) GetDialChats(ctx context.Context) ([]int64, error) {
	keys, err := s.redisClient.Keys(ctx, getAllDialChats()).Result()
	if err == redisV8.Nil {
		return []int64{}, nil
	}
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during scanning chats %v", err)
		return nil, err
	}

	var ret0 = []int64{}
	for _, dialChatKey := range keys {
		chatId, err := chatIdFromKey(dialChatKey)
		if err != nil {
			logger.GetLogEntry(ctx).Warnf("Error during parsing chatId %v", err)
			continue
		}
		ret0 = append(ret0, chatId)
	}
	return ret0, nil
}

func (s *DialRedisRepository) GetUsersToDial(ctx context.Context, chatId int64) ([]int64, error) {
	members, err := s.redisClient.SMembers(ctx, dialChatMembersKey(chatId)).Result()
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
