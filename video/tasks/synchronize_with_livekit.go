package tasks

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
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
	userService   *services.UserService
	tracer             trace.Tracer
}

func NewSynchronizeWithLivekitService(redisService *services.DialRedisRepository, userService *services.UserService) *SynchronizeWithLivekitService {
	trcr := otel.Tracer("scheduler/clean-redis-orphan-service")
	return &SynchronizeWithLivekitService{
		redisService: redisService,
		userService:  userService,
		tracer:       trcr,
	}
}

func (srv *SynchronizeWithLivekitService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.ChatDialer")
	defer span.End()

	Logger.Debugf("Invoked periodic SynchronizeWithLivekit")
	// cleanOrphans
	srv.cleanOrphans(ctx)

}

func (srv *SynchronizeWithLivekitService) cleanOrphans(ctx context.Context) {
	userIds, err := srv.redisService.GetUserIds(ctx)
	if err != nil {
		GetLogEntry(ctx).Errorf("Unable to get userCallStateKeys")
		return
	}

	// move orphaned users in "inCall" status to "cancelling"
	for _, userId := range userIds {

		userCallState, chatId, _, markedForChangeStatusAttempt, ownerId, err := srv.redisService.GetUserCallState(ctx, userId)
		if err != nil {
			GetLogEntry(ctx).Errorf("Unable to get user call state %v", err)
			continue
		}
		if services.ShouldProlong(userCallState) {
			if markedForChangeStatusAttempt >= viper.GetInt("schedulers.synchronizeWithLivekitTask.orphanUserIteration") {
				GetLogEntry(ctx).Infof("Removing userCallState of user %v to %v because attempts were exhausted", userId, services.CallStatusRemoving)
				err := srv.redisService.RemoveFromDialList(ctx, userId, true, ownerId)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable user status chatId %v, userId %v", chatId, userId)
					continue
				}
				continue
			}

			videoParticipants, err := srv.userService.GetVideoParticipants(chatId, ctx)
			if err != nil {
				GetLogEntry(ctx).Errorf("Unable to get video participants of %v", chatId)
				continue
			}
			if !utils.Contains(videoParticipants, userId) {
				newAttempt := markedForChangeStatusAttempt + 1
				GetLogEntry(ctx).Infof("Setting attempt %v on userCallState of user %v because they aren't among video room participants", newAttempt, userId)
				err = srv.redisService.SetMarkedForChangeStatusAttempt(ctx, userId, markedForChangeStatusAttempt + 1)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to set user markedForChangeStatusAttempt userId %v", userId)
					continue
				}
			} else {
				if markedForChangeStatusAttempt >= 1 {
					GetLogEntry(ctx).Infof("Resetting attempt on userCallState of user %v because they appeared among video room participants", userId)

					err = srv.redisService.SetMarkedForChangeStatusAttempt(ctx, userId, services.UserCallMarkedForOrphanRemoveAttemptNotSet)
					if err != nil {
						GetLogEntry(ctx).Errorf("Unable to set user markedForChangeStatusAttempt userId %v", userId)
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
