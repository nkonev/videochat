package services

import (
	"context"
	"errors"
	"fmt"
	redisV8 "github.com/go-redis/redis/v8"
	"nkonev.name/video/logger"
	"nkonev.name/video/utils"
	"strings"
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

const NoUser = -1

const NoChat = -1

const CallStatusInviting = "inviting" // the status scheduler should remain,
const CallStatusInCall = "inCall" // the status scheduler should remain
const CallStatusCancelling = "cancelling" // will be removed after some time automatically by scheduler

const CallStatusRemoving = "removing" // will be removed after some time automatically by scheduler

const CallStatusNotFound = ""


const UserCallStatusKey = "status"
const UserCallChatIdKey = "chatId"
const UserCallMarkedForRemoveAtKey = "markedForRemoveAt"

const UserCallMarkedForOrphanRemoveAttemptKey = "markedForOrphanRemoveAttempt"

const UserCallCallOwnerKey = "userCallOwner"

const UserCallMarkedForRemoveAtNotSet = 0

const UserCallMarkedForOrphanRemoveAttemptNotSet = 0


// aka Should be changed Automatically After Timeout
func IsTemporary(userCallStatus string) bool {
	return userCallStatus == CallStatusCancelling || userCallStatus == CallStatusRemoving
}

func ShouldProlong(userCallStatus string) bool { // all NOT IsTemporary but not CallStatusNotFound
	return userCallStatus == CallStatusInCall
}

func CanOverrideCallStatus(userCallStatus string) bool {
	return IsTemporary(userCallStatus) || userCallStatus == CallStatusNotFound
}

func (s *DialRedisRepository) AddToDialList(ctx context.Context, userId, chatId int64, ownerId int64, callStatus string) error {

	err := s.addToSet(ctx, userId, ownerId)
	if err != nil {
		return err
	}

	err = s.redisClient.HSet(ctx, dialUserCallStateKey(userId),
		UserCallStatusKey, callStatus,
		UserCallChatIdKey, chatId,
		UserCallMarkedForRemoveAtKey, UserCallMarkedForRemoveAtNotSet,
		UserCallMarkedForOrphanRemoveAttemptKey, UserCallMarkedForOrphanRemoveAttemptNotSet,
		UserCallCallOwnerKey, ownerId,
	).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial %v", err)
		return err
	}

	return nil
}

func (s *DialRedisRepository) addToSet(ctx context.Context, userId int64, ownerId int64) (error) {
	err := s.redisClient.SAdd(ctx, userOwnedCallsKey(ownerId), userId).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial %v", err)
		return err
	}
	return err
}

func (s *DialRedisRepository) GetUserCalls(ctx context.Context, ownerId int64) ([]int64, error) {
	members, err := s.redisClient.SMembers(ctx, userOwnedCallsKey(ownerId)).Result()
	if err == redisV8.Nil {
		return []int64{}, nil
	}
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during scanning callees of user %v", err)
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

func userOwnedCallsKey(userId int64) string {
	return fmt.Sprintf("dials_of_user:%v", userId)
}

func dialUserCallStateKey(userId int64) string {
	return fmt.Sprintf("user_call_state:%v", userId)
}


func allChatUserCallsKey() string {
	return "user_call_state:*"
}

func getUserOwesCalls() string {
	return "dials_of_user:*"
}

func (s *DialRedisRepository) RemoveFromDialList(ctx context.Context, userId int64, removeUserState bool, ownerId int64) error {
	// remove from "dialchat" members
	err := s.removeFromSet(ctx, userId, ownerId)
	if err != nil {
		return err
	}

	if removeUserState {
		// remove "user_call_state"
		err = s.redisClient.Del(ctx, dialUserCallStateKey(userId)).Err()
		if err != nil {
			logger.GetLogEntry(ctx).Errorf("Error during deleting dialMeta %v", err)
			return err
		}
	}

	cardinality, err := s.redisClient.SCard(ctx, userOwnedCallsKey(ownerId)).Result()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during performing SCARD %v", err)
		return err
	}
	// clean
	if cardinality == 0 {
		// remove "dialchat" on zero members
		err = s.redisClient.Del(ctx, userOwnedCallsKey(ownerId)).Err()
		if err != nil {
			logger.GetLogEntry(ctx).Errorf("Error during deleting ChatMembers %v", err)
			return err
		}
	}

	return nil
}

func (s *DialRedisRepository) removeFromSet(ctx context.Context, userId int64, ownerId int64) (error) {
	_, err := s.redisClient.SRem(ctx, userOwnedCallsKey(ownerId), userId).Result()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during performing SREM %v", err)
		return err
	}
	return nil
}

func (s *DialRedisRepository) SetUserStatus(ctx context.Context, userId int64, callStatus string) error {
	err := s.redisClient.HSet(ctx, dialUserCallStateKey(userId), UserCallStatusKey, callStatus).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial %v", err)
		return err
	}
	return nil
}

func (s *DialRedisRepository) SetCurrentTimeForRemoving(ctx context.Context, userId int64) error {
	err := s.redisClient.HSet(ctx, dialUserCallStateKey(userId), UserCallMarkedForRemoveAtKey, time.Now().UnixMilli()).Err()
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during adding user to dial %v", err)
		return err
	}
	return nil
}



func (s *DialRedisRepository) GetUsersOwesCalls(ctx context.Context) ([]int64, error) {
	keys, err := s.redisClient.Keys(ctx, getUserOwesCalls()).Result()
	if err == redisV8.Nil {
		return []int64{}, nil
	}
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during scanning chats %v", err)
		return nil, err
	}

	var ret0 = []int64{}
	for _, userDialKey := range keys {
		id, err := getUserId(userDialKey)
		if err != nil {
			logger.GetLogEntry(ctx).Warnf("Error during parsing chatId %v", err)
			continue
		}
		ret0 = append(ret0, id)
	}
	return ret0, nil
}

func (s *DialRedisRepository) GetUsersOfOwnersDial(ctx context.Context, ownerId int64) ([]int64, error) {
	members, err := s.redisClient.SMembers(ctx, userOwnedCallsKey(ownerId)).Result()
	if err == redisV8.Nil {
		return []int64{}, nil
	}
	if err != nil {
		logger.GetLogEntry(ctx).Errorf("Error during scanning users of chat %v", err)
		return nil, err
	}
	var ret = []int64{}
	for _, userIdString := range members {
		parseInt64, err := utils.ParseInt64(userIdString)
		if err != nil {
			logger.GetLogEntry(ctx).Warnf("Error during parsing userId %v", err)
			continue
		}
		ret = append(ret, parseInt64)
	}
	return ret, nil
}

func (s *DialRedisRepository) GetUserCallStatus(ctx context.Context, userId int64) (string, error) {
	 status, err := s.redisClient.HGet(ctx, dialUserCallStateKey(userId), UserCallStatusKey).Result()
	 if err == redisV8.Nil {
		 return CallStatusNotFound, nil
	 }
	 return status, err
}

func (s *DialRedisRepository) ResetExpiration(ctx context.Context, userId int64) error {
	return s.redisClient.Persist(ctx, dialUserCallStateKey(userId)).Err()
}

func (s *DialRedisRepository) ResetOwner(ctx context.Context, userId int64) error {
	return s.redisClient.HSet(ctx, dialUserCallStateKey(userId), UserCallCallOwnerKey, NoUser).Err()
}

func (s *DialRedisRepository) GetUserCallState(ctx context.Context, userId int64) (string, int64, int64, int, int64, error) {
	status, err := s.redisClient.HGetAll(ctx, dialUserCallStateKey(userId)).Result()
	if err == redisV8.Nil || len(status) == 0 {
		return CallStatusNotFound, NoChat, -1, -1, -1, nil
	}

	userCallStatus := status[UserCallStatusKey]

	chatId, err := utils.ParseInt64(status[UserCallChatIdKey])
	if err != nil {
		return CallStatusNotFound, NoChat, -1, -1, -1, err
	}

	maybeUserCallMarkedForRemoveAt, ok := status[UserCallMarkedForRemoveAtKey]
	var userCallMarkedForRemoveAt int64 = UserCallMarkedForRemoveAtNotSet
	if ok {
		userCallMarkedForRemoveAt, err = utils.ParseInt64(maybeUserCallMarkedForRemoveAt)
		if err != nil {
			return CallStatusNotFound, NoChat, -1, -1, -1, err
		}
	}

	var markedForChangeStatusAttempt int64 = UserCallMarkedForOrphanRemoveAttemptNotSet
	maybeMarkedForChangeStatusAttempt, ok := status[UserCallMarkedForOrphanRemoveAttemptKey]
	if ok {
		markedForChangeStatusAttempt, err = utils.ParseInt64(maybeMarkedForChangeStatusAttempt)
		if err != nil {
			return CallStatusNotFound, NoChat, -1, -1, -1, err
		}
	}

	var userCallCallOwner int64 = NoUser
	maybeUserCallCallOwner, ok := status[UserCallCallOwnerKey]
	if ok {
		userCallCallOwner, err = utils.ParseInt64(maybeUserCallCallOwner)
		if err != nil {
			return CallStatusNotFound, NoChat, -1, -1, -1, err
		}
	}

	return userCallStatus, chatId, userCallMarkedForRemoveAt, int(markedForChangeStatusAttempt), userCallCallOwner, nil
}

func (s *DialRedisRepository) GetUserIds(ctx context.Context) ([]int64, error) {
	var ret = make([]int64, 0)
	result, err := s.redisClient.Keys(ctx, allChatUserCallsKey()).Result()
	if err != nil {
		return ret, err
	}
	for _, userIdString := range result {
		id, err := getUserId(userIdString)
		if err != nil {
			logger.GetLogEntry(ctx).Errorf("Error during converting userId %v", userIdString)
			continue
		}
		ret = append(ret, id)
	}
	return ret, nil
}

func (s *DialRedisRepository) SetMarkedForChangeStatusAttempt(ctx context.Context, userId int64, markedForChangeStatusAttempt int) error {
	return s.redisClient.HSet(ctx, dialUserCallStateKey(userId), UserCallMarkedForOrphanRemoveAttemptKey, markedForChangeStatusAttempt).Err()
}

// side effect: it removes inVideoUsers (except leavingOwner) from leavingOwner's dials_of_user:1
func (s *DialRedisRepository) TransferOwnership(ctx context.Context, inVideoUsers []int64, leavingOwner int64, chatId int64) error {
	wasTransferred := false
	wasErrored := false
	if len(inVideoUsers) > 0 {
		oppositeUser := utils.GetOppositeUser(inVideoUsers, leavingOwner)

		if oppositeUser != nil {
			oppositeUserStatus, err := s.GetUserCallStatus(ctx, *oppositeUser)
			if err != nil {
				logger.GetLogEntry(ctx).Errorf("Error getting opposite user status %v", err)
				return err
			}

			if !IsTemporary(oppositeUserStatus) {
				newOwner := *oppositeUser

				logger.GetLogEntry(ctx).Infof("Transfering ownership over chat %v from user %v to user %v", chatId, leavingOwner, newOwner)

				for _, userId := range inVideoUsers {
					if userId != leavingOwner {

						err := s.addToSet(ctx, userId, newOwner)
						if err != nil {
							logger.GetLogEntry(ctx).Errorf("Error during changing owner %v", err)
							wasErrored = true
						}

						err = s.redisClient.HSet(ctx, dialUserCallStateKey(userId), UserCallCallOwnerKey, newOwner).Err()
						if err != nil {
							logger.GetLogEntry(ctx).Errorf("Error during adding user to dial %v", err)
							wasErrored = true
						}

						err = s.removeFromSet(ctx, userId, leavingOwner)
						if err != nil {
							logger.GetLogEntry(ctx).Errorf("Error during removing user from previous dial list %v", err)
							wasErrored = true
						}

						wasTransferred = true
					}
				}

			}
		}
	}
	if !wasTransferred {
		logger.GetLogEntry(ctx).Infof("Unable to transfer ownership over chat %v from user %v because there is no candidates. All the metadata is going to be removed automatically after a while", chatId, leavingOwner)
	}
	if wasErrored {
		return errors.New("Some error occurred during TransferOwnership")
	}
	return nil
}


func getUserId(userCallStateKey string) (int64, error) {
	split := strings.Split(userCallStateKey, ":")
	if len(split) != 2 {
		return -1, fmt.Errorf("Wrong split lenght of %v, expected 2", userCallStateKey)
	}
	return utils.ParseInt64(split[1])
}
