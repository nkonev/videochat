package tasks

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/video/config"
	"nkonev.name/video/db"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
)

type UsersInVideoStatusNotifierService struct {
	scheduleService *services.StateChangedEventService
	conf            *config.ExtendedConfig
	tracer          trace.Tracer
	database        *db.DB
}

func NewUsersInVideoStatusNotifierService(scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig, database *db.DB) *UsersInVideoStatusNotifierService {
	trcr := otel.Tracer("scheduler/users-in-video-notifier")
	return &UsersInVideoStatusNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
		tracer:          trcr,
		database:        database,
	}
}

func (srv *UsersInVideoStatusNotifierService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.UsersInVideoStatusNotifier")
	defer span.End()

	GetLogEntry(ctx).Debugf("Invoked periodic UsersInVideoStatusNotifier")

	err := db.Transact(srv.database, func(tx *db.Tx) error {
		srv.scheduleService.NotifyAllChatsAboutUsersInVideoStatus(ctx,  tx,nil)
		return nil
	})
	if err != nil {
		GetLogEntry(ctx).Errorf("error during invoking NotifyAllChatsAboutUsersInVideoStatus in transaction: %v", err)
	}

	GetLogEntry(ctx).Debugf("End of UsersInVideoStatusNotifier")
}

type UsersInVideoStatusNotifierTask struct {
	*gointerlock.GoInterval
}

func UsersInVideoStatusNotifierScheduler(
	redisConnector *redisV8.Client,
	service *UsersInVideoStatusNotifierService,
	conf *config.ExtendedConfig,
) *UsersInVideoStatusNotifierTask {
	var interv = viper.GetDuration("schedulers.usersInVideoStatusNotifierTask.notificationPeriod")
	Logger.Infof("Created users video status periodic notificator with interval %v", interv)
	return &UsersInVideoStatusNotifierTask{&gointerlock.GoInterval{
		Name:           "usersInVideoStatusNotifierTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
