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
)

type VideoCallUsersCountNotifierService struct {
	scheduleService *services.StateChangedEventService
	conf            *config.ExtendedConfig
	tracer             trace.Tracer
}

func NewVideoCallUsersCountNotifierService(scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig) *VideoCallUsersCountNotifierService {
	trcr := otel.Tracer("scheduler/video-call-users-count-notifier")
	return &VideoCallUsersCountNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
		tracer:          trcr,
	}
}

func (srv *VideoCallUsersCountNotifierService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.VideoCallUsersCountNotifier")
	defer span.End()

	Logger.Debugf("Invoked periodic ChatNotifier")
	srv.scheduleService.NotifyAllChatsAboutVideoCallUsersCount(ctx)

	Logger.Debugf("End of ChatNotifier")
}

type VideoCallUsersCountNotifierTask struct {
	*gointerlock.GoInterval
}

func VideoCallUsersCountNotifierScheduler(
	redisConnector *redisV8.Client,
	service *VideoCallUsersCountNotifierService,
	conf *config.ExtendedConfig,
) *VideoCallUsersCountNotifierTask {
	var interv = viper.GetDuration("schedulers.videoCallUsersCountNotifierTask.notificationPeriod")
	Logger.Infof("Created video call users count periodic notificator with interval %v", interv)
	return &VideoCallUsersCountNotifierTask{&gointerlock.GoInterval{
		Name:           "videoCallUsersCountNotifierTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
