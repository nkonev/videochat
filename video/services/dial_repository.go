package services

import (
	"context"
	"fmt"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
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

const ownerIdConstant = "ownerId" // also it is dial owner - person who starts the dial

const NoUser = -1

const CallStatusInviting = "inviting"
const CallStatusInCall = "inCall"

const UserCallStatusKey = "status"
const UserCallChatIdKey = "chatId"



func (s *DialRedisRepository) AddToDialList(ctx context.Context, userId, chatId int64, ownerId int64) error {
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

	err = s.redisClient.HSet(ctx, dialChatUserCallsKey(userId), UserCallStatusKey, CallStatusInviting, UserCallChatIdKey, chatId).Err()
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

func dialChatUserCallsKey(chatId int64) string {
	return fmt.Sprintf("user_call_state:%v", chatId)
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

// TODO here and in RemoveAllDials() not to remove immediately, but switch status and set expiration on user_call_status key equal to "room is closing" timeout
func (s *DialRedisRepository) RemoveFromDialList(ctx context.Context, userId, chatId int64) error {
	_, err := s.redisClient.SRem(ctx, dialChatMembersKey(chatId), userId).Result() // TODO instead of it - just change status on "canceling"
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during performing SREM %v", err)
		return err
	}

	card, err := s.redisClient.SCard(ctx, dialChatMembersKey(chatId)).Result()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during performing SCARD %v", err)
		return err
	}
	if card == 0 { // TODO create run periodically task, which checks if there are any active calls (dialChatMembersKey) by chatId and if zero - then remove the entire construction
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

		// remove "user_call_state" on zero members
		err = s.redisClient.Del(ctx, dialChatUserCallsKey(userId)).Err()
		if err != nil {
			logger.GetLogEntry(ctx).Errorf("Error during deleting dialMeta %v", err)
			return err
		}

	}
	return nil
}

// aka remove all dials or close call
func (s *DialRedisRepository) RemoveAllDials(ctx context.Context, chatId int64) {
	usersOfDial, err := s.GetUsersToDial(ctx, chatId)
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error %v", err)
		return
	}

	// remove "dialchat"
	err = s.redisClient.Del(ctx, dialChatMembersKey(chatId)).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during performing SREM %v", err)
		return
	}

	// remove "dialmeta"
	err = s.redisClient.Del(ctx, dialMetaKey(chatId)).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during deleting dialMeta %v", err)
		return
	}

	// remove "user_call_state"
	for _, userId := range usersOfDial {
		err = s.redisClient.Del(ctx, dialChatUserCallsKey(userId)).Err()
		if err != nil {
			logger.GetLogEntry(ctx).Errorf("Error during deleting dialMeta %v", err)
		}
	}
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
