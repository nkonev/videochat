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

type UsersVideoStatusNotifierService struct {
	scheduleService *services.StateChangedEventService
	conf            *config.ExtendedConfig
	tracer             trace.Tracer
}

func NewUsersVideoStatusNotifierService(scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig) *UsersVideoStatusNotifierService {
	trcr := otel.Tracer("scheduler/new-users-video-notifier")
	return &UsersVideoStatusNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
		tracer:          trcr,
	}
}

func (srv *UsersVideoStatusNotifierService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.UsersVideoStatusNotifier")
	defer span.End()

	Logger.Debugf("Invoked periodic UsersVideoStatusNotifier")
	srv.scheduleService.NotifyAllChatsAboutUsersVideoStatus(ctx)

	Logger.Debugf("End of UsersVideoStatusNotifier")
}

type UsersVideoStatusNotifierTask struct {
	*gointerlock.GoInterval
}

func UsersVideoStatusNotifierScheduler(
	redisConnector *redisV8.Client,
	service *UsersVideoStatusNotifierService,
	conf *config.ExtendedConfig,
) *UsersVideoStatusNotifierTask {
	var interv = viper.GetDuration("schedulers.usersVideoStatusNotifierTask.notificationPeriod")
	Logger.Infof("Created users video status periodic notificator with interval %v", interv)
	return &UsersVideoStatusNotifierTask{&gointerlock.GoInterval{
		Name:           "usersVideoStatusNotifierTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
