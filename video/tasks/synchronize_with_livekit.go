package tasks

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
)

type SynchronizeWithLivekitService struct {
	redisService            *services.DialRedisRepository
	userService   			*services.UserService
	tracer             		trace.Tracer
	livekitRoomClient   	*lksdk.RoomServiceClient
}

func NewSynchronizeWithLivekitService(redisService *services.DialRedisRepository, userService *services.UserService, livekitRoomClient *lksdk.RoomServiceClient) *SynchronizeWithLivekitService {
	trcr := otel.Tracer("scheduler/clean-redis-orphan-service")
	return &SynchronizeWithLivekitService{
		redisService: redisService,
		userService:  userService,
		tracer:       trcr,
		livekitRoomClient: livekitRoomClient,
	}
}

func (srv *SynchronizeWithLivekitService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.ChatDialer")
	defer span.End()

	Logger.Debugf("Invoked periodic SynchronizeWithLivekit")

	userIds, err := srv.redisService.GetUserIds(ctx)
	if err != nil {
		GetLogEntry(ctx).Errorf("Unable to get userCallStateKeys")
		return
	}

	srv.cleanOrphans(ctx, userIds)

	srv.createParticipants(ctx, userIds)
}

func (srv *SynchronizeWithLivekitService) cleanOrphans(ctx context.Context, userIds []int64) {

	// move orphaned users in "inCall" status to "cancelling"
	for _, userId := range userIds {

		userCallState, chatId, _, markedForChangeStatusAttempt, ownerId, err := srv.redisService.GetUserCallState(ctx, userId)
		if err != nil {
			GetLogEntry(ctx).Errorf("Unable to get user call state %v", err)
			continue
		}
		if services.ShouldProlong(userCallState) { // consider only users, hanged in "inCall" state in redis and not presented in livekit
			// removing
			if markedForChangeStatusAttempt >= viper.GetInt("schedulers.synchronizeWithLivekitTask.orphanUserIteration") {
				GetLogEntry(ctx).Warnf("Removing userCallState of user %v to %v because attempts were exhausted", userId, services.CallStatusRemoving)
				err := srv.redisService.RemoveFromDialList(ctx, userId, true, ownerId)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable user status chatId %v, userId %v", chatId, userId)
					continue
				}
				continue
			}

			// changing attempt number
			videoParticipants, err := srv.userService.GetVideoParticipants(chatId, ctx)
			if err != nil {
				GetLogEntry(ctx).Errorf("Unable to get video participants of %v", chatId)
				continue
			}
			if !utils.Contains(videoParticipants, userId) {
				newAttempt := markedForChangeStatusAttempt + 1
				GetLogEntry(ctx).Infof("Setting attempt %v on userCallState %v of user %v because they aren't among video room participants", newAttempt, userCallState, userId)
				err = srv.redisService.SetMarkedForChangeStatusAttempt(ctx, userId, markedForChangeStatusAttempt + 1)
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

func (srv *SynchronizeWithLivekitService) createParticipants(ctx context.Context, userIds []int64) {
	listRoomReq := &livekit.ListRoomsRequest{}
	rooms, err := srv.livekitRoomClient.ListRooms(ctx, listRoomReq)
	if err != nil {
		Logger.Error(err, "error during reading rooms %v", err)
		return
	}

	for _, room := range rooms.Rooms {
		chatId, err := utils.GetRoomIdFromName(room.Name)
		if err != nil {
			Logger.Errorf("got error during getting chat id from roomName %v %v", room.Name, err)
			continue
		}

		videoParticipants, err := srv.userService.GetVideoParticipants(chatId, ctx)
		if err != nil {
			Logger.Errorf("got error during getting videoParticipants from roomName %v %v", room.Name, err)
			continue
		}

		// if no such users
		for _, videoUserId := range videoParticipants {
			userCallState, _, _, _, _, err := srv.redisService.GetUserCallState(ctx, videoUserId)
			if err != nil {
				GetLogEntry(ctx).Errorf("Unable to get user call state %v", err)
				continue
			}

			// if there is no status in redis, but we have it in livekit - then create
			if userCallState == services.CallStatusNotFound {
				if !utils.Contains(userIds, videoUserId) {
					GetLogEntry(ctx).Warnf("Populating user %v from livekit to redis in chat %v", videoUserId, chatId)
					err = srv.redisService.AddToDialList(ctx, videoUserId, chatId, videoUserId, services.CallStatusInCall)
					if err != nil {
						GetLogEntry(ctx).Errorf("Unable to AddToDialList %v", err)
						continue
					}
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
