package tasks

import (
	"context"
	"github.com/nkonev/dcron"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/video/config"
	"nkonev.name/video/logger"
	"nkonev.name/video/services"
)

type VideoCallUsersCountNotifierService struct {
	scheduleService *services.StateChangedEventService
	conf            *config.ExtendedConfig
	tracer          trace.Tracer
	lgr             *logger.Logger
}

func NewVideoCallUsersCountNotifierService(lgr *logger.Logger, scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig) *VideoCallUsersCountNotifierService {
	trcr := otel.Tracer("scheduler/video-call-users-count-notifier")
	return &VideoCallUsersCountNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
		tracer:          trcr,
		lgr:             lgr,
	}
}

func (srv *VideoCallUsersCountNotifierService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.VideoCallUsersCountNotifier")
	defer span.End()

	srv.lgr.WithTracing(ctx).Debugf("Invoked periodic ChatNotifier")
	srv.scheduleService.NotifyAllChatsAboutVideoCallUsersCount(ctx)

	srv.lgr.WithTracing(ctx).Debugf("End of ChatNotifier")
}

type VideoCallUsersCountNotifierTask struct {
	dcron.Job
}

func VideoCallUsersCountNotifierScheduler(
	lgr *logger.Logger,
	service *VideoCallUsersCountNotifierService,
) *VideoCallUsersCountNotifierTask {
	const key = "videoCallUsersCountNotifierTask"
	var str = viper.GetString("schedulers." + key + ".cron")
	lgr.Infof("Created VideoCallUsersCountNotifierScheduler with cron %v", str)

	job := dcron.NewJob(key, str, func(ctx context.Context) error {
		service.doJob()
		return nil
	})

	return &VideoCallUsersCountNotifierTask{job}
}
