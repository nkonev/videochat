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

type RecordingNotifierService struct {
	scheduleService *services.StateChangedEventService
	conf            *config.ExtendedConfig
	tracer             trace.Tracer
}

func NewRecordingNotifierService(scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig) *RecordingNotifierService {
	trcr := otel.Tracer("scheduler/new-recording-notifier")
	return &RecordingNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
		tracer: trcr,
	}
}

func (srv *RecordingNotifierService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.RecordingNotifier")
	defer span.End()

	Logger.Debugf("Invoked periodic RecordingNotifierService")
	srv.scheduleService.NotifyAllChatsAboutVideoCallRecording(ctx)

	Logger.Debugf("End of RecordingNotifierService")
}

type RecordingNotifierTask struct {
	*gointerlock.GoInterval
}

func RecordingNotifierScheduler(
	redisConnector *redisV8.Client,
	service *VideoCallUsersCountNotifierService,
	conf *config.ExtendedConfig,
) *RecordingNotifierTask {
	var interv = viper.GetDuration("schedulers.videoRecordingNotifierTask.notificationPeriod")
	Logger.Infof("Created RecordingNotifierService periodic notificator with interval %v", interv)
	return &RecordingNotifierTask{&gointerlock.GoInterval{
		Name:           "videoRecordingNotifierTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
