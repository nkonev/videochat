package tasks

import (
	"context"
	"errors"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
	"strconv"
)

var numErr = &strconv.NumError{}

type SynchronizeWithLivekitService struct {
	redisService            *services.DialRedisRepository
	userService   			*services.UserService
	tracer             		trace.Tracer
	livekitRoomClient   	*lksdk.RoomServiceClient
	restClient              *client.RestClient
}

func NewSynchronizeWithLivekitService(redisService *services.DialRedisRepository, userService *services.UserService, livekitRoomClient *lksdk.RoomServiceClient, restClient *client.RestClient) *SynchronizeWithLivekitService {
	trcr := otel.Tracer("scheduler/synchronize-with-livekit")
	return &SynchronizeWithLivekitService{
		redisService: redisService,
		userService:  userService,
		tracer:       trcr,
		livekitRoomClient: livekitRoomClient,
		restClient: restClient,
	}
}

func (srv *SynchronizeWithLivekitService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.ChatDialer")
	defer span.End()

	GetLogEntry(ctx).Debugf("Invoked periodic SynchronizeWithLivekit")

	allUserIds, err := srv.redisService.GetAllUserIds(ctx)
	if err != nil {
		GetLogEntry(ctx).Errorf("Unable to get userCallStateKeys")
		return
	}

	srv.cleanOrphans(ctx, allUserIds)

	srv.createParticipants(ctx)
}

func (srv *SynchronizeWithLivekitService) cleanOrphans(ctx context.Context, userIds []int64) {

	// move orphaned users in "inCall" status to "cancelling"
	for _, userId := range userIds {
		userCallState, chatId, _, markedForChangeStatusAttempt, ownerId, _, _, err := srv.redisService.GetUserCallState(ctx, userId)
		if err != nil {
			GetLogEntry(ctx).Errorf("Unable to get user call state %v", err)

			if isNonRestorableError(err) {
				GetLogEntry(ctx).Warnf("Going to remove invalid call state %v", err)
				err := srv.redisService.RemoveFromDialList(ctx, userId, true, ownerId)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable invoke RemoveFromDialList, user %v, error %v", userId, err)
					continue
				}
			}

			continue
		}

		if services.ShouldProlong(userCallState) { // consider only users, hanged in "inCall" state in redis and not presented in livekit
			// removing
			if markedForChangeStatusAttempt >= viper.GetInt("schedulers.synchronizeWithLivekitTask.orphanUserIteration") {
				// user is owner of call
				GetLogEntry(ctx).Warnf("Removing owned call by user userId %v because attempts were exhausted", userId)
				err = srv.redisService.RemoveOwn(ctx, userId)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to remove owned call by user userId %v, chatId %v, error %v", userId, chatId, err)
				}

				// user is own by somebody
				GetLogEntry(ctx).Warnf("Removing userCallState of user %v, owned by ownerId %v because attempts were exhausted", userId, ownerId)
				err := srv.redisService.RemoveFromDialList(ctx, userId, true, ownerId)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to remove user userId %v owned by ownerId %v, chatId %v, error %v", userId, ownerId, chatId, err)
				}
				continue // because we don't need increment an attempt
			}

			// changing attempt number
			videoParticipants, err := srv.userService.GetVideoParticipants(ctx, chatId)
			if err != nil {
				GetLogEntry(ctx).Errorf("Unable to get video participants of %v", chatId)
				continue
			}
			if !utils.Contains(videoParticipants, userId) {
				newAttempt := markedForChangeStatusAttempt + 1
				GetLogEntry(ctx).Infof("Setting attempt %v on userCallState %v of user %v because they aren't among video room participants", newAttempt, userCallState, userId)
				err = srv.redisService.SetMarkedForChangeStatusAttempt(ctx, userId, newAttempt)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to set user markedForChangeStatusAttempt userId %v", userId)
					continue
				}
			} else {
				if markedForChangeStatusAttempt >= 1 {
					GetLogEntry(ctx).Infof("Resetting attempt on userCallState %v of user %v because they appeared among video room participants", userCallState, userId)

					err = srv.redisService.SetMarkedForChangeStatusAttempt(ctx, userId, services.UserCallMarkedForOrphanRemoveAttemptNotSet)
					if err != nil {
						GetLogEntry(ctx).Errorf("Unable to set user markedForChangeStatusAttempt userId %v", userId)
						continue
					}
				}
			}

		} // else branch not needed, because they removed from chat_dialer task's cleanNotNeededAnymoreDialRedisData()
	}
}

func isNonRestorableError(err error) bool {
	return errors.As(err, &numErr)
}

func (srv *SynchronizeWithLivekitService) createParticipants(ctx context.Context) {
	listRoomReq := &livekit.ListRoomsRequest{}
	rooms, err := srv.livekitRoomClient.ListRooms(ctx, listRoomReq)
	if err != nil {
		GetLogEntry(ctx).Error(err, "error during reading rooms %v", err)
		return
	}

	for _, room := range rooms.Rooms {
		chatId, err := utils.GetRoomIdFromName(room.Name)
		if err != nil {
			GetLogEntry(ctx).Errorf("got error during getting chat id from roomName %v %v", room.Name, err)
			continue
		}

		videoParticipants, err := srv.userService.GetVideoParticipants(ctx, chatId)
		if err != nil {
			GetLogEntry(ctx).Errorf("got error during getting videoParticipants from roomName %v %v", room.Name, err)
			continue
		}

		// if no such users
		for _, videoUserId := range videoParticipants {
			userCallState, _, _, _, _, _, _, err := srv.redisService.GetUserCallState(ctx, videoUserId)
			if err != nil {
				GetLogEntry(ctx).Errorf("Unable to get user call state %v", err)
				continue
			}

			// if there is no status in redis, but we have it in livekit - then create
			if userCallState == services.CallStatusNotFound {
				GetLogEntry(ctx).Warnf("Populating user %v from livekit to redis in chat %v", videoUserId, chatId)

				chatInfo, err := srv.restClient.GetBasicChatInfo(ctx, chatId, videoUserId)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to GetBasicChatInfo %v", err)
					continue
				}

				aaaUsers, err := srv.restClient.GetUsers(ctx, []int64{videoUserId})
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to users %v", err)
					continue
				}
				if len(aaaUsers) != 1 {
					GetLogEntry(ctx).Errorf("len of users is %v, but we need 1", len(aaaUsers))
					continue
				}
				aaaUser := aaaUsers[0]

				err = srv.redisService.AddToDialList(ctx, videoUserId, chatId, videoUserId, services.CallStatusInCall, utils.NullToEmpty(aaaUser.Avatar), chatInfo.TetATet) // dummy set NoAvatar
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to AddToDialList %v", err)
					continue
				}
			}
		}
	}
}


type SynchronizeWithLivekitTask struct {
	*gointerlock.GoInterval
}

func SynchronizeWithLivekitSheduler(
	redisConnector *redisV8.Client,
	service *SynchronizeWithLivekitService,
	conf *config.ExtendedConfig,
) *SynchronizeWithLivekitTask {
	var interv = viper.GetDuration("schedulers.synchronizeWithLivekitTask.period")
	Logger.Infof("Synchronize with livekit task with interval %v", interv)
	return &SynchronizeWithLivekitTask{&gointerlock.GoInterval{
		Name:           "synchronizeWithLivekitTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
