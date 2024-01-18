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

type CleanOrphanRedisEntriesService struct {
	redisService            *services.DialRedisRepository
	userService   *services.UserService
	tracer             trace.Tracer
}

func NewCleanOrphanRedisEntriesService(redisService *services.DialRedisRepository, userService *services.UserService) *CleanOrphanRedisEntriesService {
	trcr := otel.Tracer("scheduler/clean-redis-orphan-service")
	return &CleanOrphanRedisEntriesService{
		redisService: redisService,
		userService:  userService,
		tracer:       trcr,
	}
}

func (srv *CleanOrphanRedisEntriesService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.ChatDialer")
	defer span.End()

	Logger.Debugf("Invoked periodic CleanOrphanRedisEntries")
	// cleanOrphans
	srv.cleanOrphans(ctx)

}

func (srv *CleanOrphanRedisEntriesService) cleanOrphans(ctx context.Context) {
	userIds, err := srv.redisService.GetUserIds(ctx)
	if err != nil {
		GetLogEntry(ctx).Errorf("Unable to get userCallStateKeys")
		return
	}

	// move orphaned users in "inCall" status to "cancelling"
	for _, userId := range userIds {

		userCallState, chatId, _, markedForChangeStatusAttempt, _, err := srv.redisService.GetUserCallState(ctx, userId)
		if err != nil {
			GetLogEntry(ctx).Errorf("Unable to get user call state %v", err)
			continue
		}
		if services.ShouldProlong(userCallState) {
			if markedForChangeStatusAttempt >= viper.GetInt("schedulers.cleanOrphanRedisEntriesTask.orphanUserIteration") {
				GetLogEntry(ctx).Infof("Removing userCallState of user %v to %v because attempts were exhausted", userId, services.CallStatusRemoving)
				err := srv.redisService.RemoveFromDialList(ctx, userId, chatId)
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


type CleanOrphanRedisEntriesTask struct {
	*gointerlock.GoInterval
}

func CleanOrphanRedisEntriesScheduler(
	redisConnector *redisV8.Client,
	service *CleanOrphanRedisEntriesService,
	conf *config.ExtendedConfig,
) *CleanOrphanRedisEntriesTask {
	var interv = viper.GetDuration("schedulers.cleanOrphanRedisEntriesTask.period")
	Logger.Infof("Created clean orphan redis entries task with interval %v", interv)
	return &CleanOrphanRedisEntriesTask{&gointerlock.GoInterval{
		Name:           "cleanOrphanRedisEntriesTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
