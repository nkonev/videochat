package tasks

import (
	"context"
	"github.com/nkonev/dcron"
	log "github.com/sirupsen/logrus"
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
	lgr             *log.Logger
}

func NewUsersInVideoStatusNotifierService(scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig, database *db.DB, lgr *log.Logger) *UsersInVideoStatusNotifierService {
	trcr := otel.Tracer("scheduler/users-in-video-notifier")
	return &UsersInVideoStatusNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
		tracer:          trcr,
		database:        database,
		lgr:             lgr,
	}
}

func (srv *UsersInVideoStatusNotifierService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.UsersInVideoStatusNotifier")
	defer span.End()

	GetLogEntry(ctx, srv.lgr).Debugf("Invoked periodic UsersInVideoStatusNotifier")

	err := db.Transact(ctx, srv.database, func(tx *db.Tx) error {
		srv.scheduleService.NotifyAllChatsAboutUsersInVideoStatus(ctx, tx, nil)
		return nil
	})
	if err != nil {
		GetLogEntry(ctx, srv.lgr).Errorf("error during invoking NotifyAllChatsAboutUsersInVideoStatus in transaction: %v", err)
	}

	GetLogEntry(ctx, srv.lgr).Debugf("End of UsersInVideoStatusNotifier")
}

type UsersInVideoStatusNotifierTask struct {
	dcron.Job
}

func UsersInVideoStatusNotifierScheduler(
	service *UsersInVideoStatusNotifierService,
	lgr *log.Logger,
) *UsersInVideoStatusNotifierTask {
	const key = "usersInVideoStatusNotifierTask"
	var str = viper.GetString("schedulers." + key + ".cron")
	lgr.Infof("Created UsersInVideoStatusNotifierScheduler with cron %v", str)

	job := dcron.NewJob(key, str, func(ctx context.Context) error {
		service.doJob()
		return nil
	})

	return &UsersInVideoStatusNotifierTask{job}
}
