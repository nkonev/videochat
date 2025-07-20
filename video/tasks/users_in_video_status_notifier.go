package tasks

import (
	"context"
	"github.com/nkonev/dcron"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/video/config"
	"nkonev.name/video/db"
	"nkonev.name/video/logger"
	"nkonev.name/video/services"
)

type UsersInVideoStatusNotifierService struct {
	scheduleService *services.StateChangedEventService
	conf            *config.ExtendedConfig
	tracer          trace.Tracer
	database        *db.DB
	lgr             *logger.Logger
}

func NewUsersInVideoStatusNotifierService(scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig, database *db.DB, lgr *logger.Logger) *UsersInVideoStatusNotifierService {
	trcr := otel.Tracer("scheduler/users-in-video-notifier")
	return &UsersInVideoStatusNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
		tracer:          trcr,
		database:        database,
		lgr:             lgr,
	}
}

func (srv *UsersInVideoStatusNotifierService) doJob(ctx context.Context) {
	srv.lgr.WithTracing(ctx).Debugf("Invoked periodic UsersInVideoStatusNotifier")

	err := db.Transact(ctx, srv.database, func(tx *db.Tx) error {
		srv.scheduleService.NotifyAllChatsAboutUsersInVideoStatus(ctx, tx, nil)
		return nil
	})
	if err != nil {
		srv.lgr.WithTracing(ctx).Errorf("error during invoking NotifyAllChatsAboutUsersInVideoStatus in transaction: %v", err)
	}

	srv.lgr.WithTracing(ctx).Debugf("End of UsersInVideoStatusNotifier")
}

func (srv *UsersInVideoStatusNotifierService) spanStarter(ctx context.Context) (context.Context, any) {
	return srv.tracer.Start(ctx, "scheduler.UsersInVideoStatusNotifier")
}

func (srv *UsersInVideoStatusNotifierService) spanFinisher(ctx context.Context, span any) {
	span.(trace.Span).End()
}

type UsersInVideoStatusNotifierTask struct {
	dcron.Job
}

func UsersInVideoStatusNotifierScheduler(
	service *UsersInVideoStatusNotifierService,
	lgr *logger.Logger,
) *UsersInVideoStatusNotifierTask {
	const key = "usersInVideoStatusNotifierTask"
	var str = viper.GetString("schedulers." + key + ".cron")
	lgr.Infof("Created UsersInVideoStatusNotifierScheduler with cron %v", str)

	job := dcron.NewJob(key, str, func(ctx context.Context) error {
		service.doJob(ctx)
		return nil
	}, dcron.WithTracing(service.spanStarter, service.spanFinisher))

	return &UsersInVideoStatusNotifierTask{job}
}
