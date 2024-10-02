package tasks

import (
	"context"
	"github.com/nkonev/dcron"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
)

type RecordingNotifierService struct {
	scheduleService *services.StateChangedEventService
	conf            *config.ExtendedConfig
	tracer          trace.Tracer
}

func NewRecordingNotifierService(scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig) *RecordingNotifierService {
	trcr := otel.Tracer("scheduler/recording-notifier")
	return &RecordingNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
		tracer:          trcr,
	}
}

func (srv *RecordingNotifierService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.RecordingNotifier")
	defer span.End()

	GetLogEntry(ctx).Debugf("Invoked periodic RecordingNotifierService")
	srv.scheduleService.NotifyAllChatsAboutVideoCallRecording(ctx)

	GetLogEntry(ctx).Debugf("End of RecordingNotifierService")
}

type RecordingNotifierTask struct {
	dcron.Job
}

func RecordingNotifierScheduler(
	service *RecordingNotifierService,
) *RecordingNotifierTask {
	const key = "videoRecordingNotifierTask"
	var str = viper.GetString("schedulers." + key + ".cron")
	Logger.Infof("Created RecordingNotifierScheduler with cron %v", str)

	job := dcron.NewJob(key, str, func(ctx context.Context) error {
		service.doJob()
		return nil
	})

	return &RecordingNotifierTask{job}
}
