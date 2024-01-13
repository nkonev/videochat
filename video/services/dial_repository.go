package services

import (
	"context"
	"fmt"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"nkonev.name/video/logger"
	"nkonev.name/video/utils"
	"time"
)

type DialRedisRepository struct {
	redisClient *redisV8.Client
}

func NewDialRedisRepository(redisClient *redisV8.Client) *DialRedisRepository {
	return &DialRedisRepository{
		redisClient: redisClient,
	}
}

const ownerIdConstant = "ownerId" // also it is dial owner - person who starts the dial

const NoUser = -1

const CallStatusInviting = "inviting" // the status scheduler should remain,
const CallStatusInCall = "inCall" // the status scheduler should remain
const CallStatusCancelling = "cancelling" // will be removed after some time automatically by scheduler

const CallStatusRemoving = "removing" // will be removed after some time automatically by scheduler

const CallStatusNotFound = ""


const UserCallStatusKey = "status"
const UserCallChatIdKey = "chatId"
const UserCallMarkedForRemoveAtKey = "markedForRemoveAt"


func ShouldRemoveAutomaticallyAfterTimeout(userCallStatus string) bool {
	return userCallStatus == CallStatusCancelling || userCallStatus == CallStatusRemoving
}

func ShouldProlong(userCallStatus string) bool {
	return userCallStatus == CallStatusInCall
}

func CanOverrideCallStatus(userCallStatus string) bool {
	return userCallStatus == CallStatusCancelling || userCallStatus == CallStatusRemoving || userCallStatus == CallStatusNotFound
}

func (s *DialRedisRepository) AddToDialList(ctx context.Context, userId, chatId int64, ownerId int64, callStatus string) error {
	expiration := viper.GetDuration("dialExpire")

	err := s.redisClient.SAdd(ctx, dialChatMembersKey(chatId), userId).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial %v", err)
		return err
	}
	_, err = s.redisClient.Expire(ctx, dialChatMembersKey(chatId), expiration).Result()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial expiration %v", err)
		return err
	}

	err = s.redisClient.HSet(ctx, dialMetaKey(chatId), ownerIdConstant, ownerId).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during setting dial metadata %v", err)
		return err
	}
	_, err = s.redisClient.Expire(ctx, dialMetaKey(chatId), expiration).Result()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during setting dial metadata expiration %v", err)
		return err
	}

	err = s.redisClient.HSet(ctx, dialChatUserCallsKey(userId), UserCallStatusKey, callStatus, UserCallChatIdKey, chatId).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial %v", err)
		return err
	}
	_, err = s.redisClient.Expire(ctx, dialChatUserCallsKey(userId), expiration).Result()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial expiration %v", err)
		return err
	}

	return nil
}

func dialChatMembersKey(chatId int64) string {
	return fmt.Sprintf("dialchat:%v", chatId)
}

func dialChatUserCallsKey(userId int64) string {
	return fmt.Sprintf("user_call_state:%v", userId)
}

func getAllDialChats() string {
	return "dialchat:*"
}

func dialMetaKey(chatId int64) string {
	return fmt.Sprintf("dialmeta:%v", chatId)
}

func chatIdFromKey(key string) (int64, error) {
	var str string
	_, err := fmt.Sscanf(key, "dialchat:%s", &str)
	if err != nil {
		return 0, err
	}
	parseInt64, err := utils.ParseInt64(str)
	if err != nil {
		return 0, err
	}
	return parseInt64, nil
}

func (s *DialRedisRepository) RemoveFromDialList(ctx context.Context, userId, chatId int64) error {
	_, err := s.redisClient.SRem(ctx, dialChatMembersKey(chatId), userId).Result()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during performing SREM %v", err)
		return err
	}

	// remove "user_call_state" on zero members
	err = s.redisClient.Del(ctx, dialChatUserCallsKey(userId)).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during deleting dialMeta %v", err)
		return err
	}

	return nil
}

func (s *DialRedisRepository) SetUserStatus(ctx context.Context, userId int64, callStatus string) error {
	err := s.redisClient.HSet(ctx, dialChatUserCallsKey(userId), UserCallStatusKey, callStatus).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial %v", err)
		return err
	}
	return nil
}

func (s *DialRedisRepository) SetCurrentTimeForCancellation(ctx context.Context, userId int64) error {
	err := s.redisClient.HSet(ctx, dialChatUserCallsKey(userId), UserCallMarkedForRemoveAtKey, time.Now().UnixMilli()).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial %v", err)
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

// returns call's owner
func (s *DialRedisRepository) GetDialMetadata(ctx context.Context, chatId int64) (int64, error) {
	val, err := s.redisClient.HGetAll(ctx, dialMetaKey(chatId)).Result()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during getting dial metadata %v", err)
		return 0, err
	}
	if len(val) == 0 {
		return NoUser, nil
	}

	parsedOwnerId, err := utils.ParseInt64(val[ownerIdConstant])
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during parsing userId %v", err)
		return 0, err
	}
	return parsedOwnerId, nil
}

func (s *DialRedisRepository) GetUserCallStatus(ctx context.Context, userId int64) (string, error) {
	 status, err := s.redisClient.HGet(ctx, dialChatUserCallsKey(userId), UserCallStatusKey).Result()
	 if err == redisV8.Nil {
		 return CallStatusNotFound, nil
	 }
	 return status, err
}

func (s *DialRedisRepository) ResetExpiration(ctx context.Context, userId int64) error {
	return s.redisClient.Persist(ctx, dialChatUserCallsKey(userId)).Err()
}
