package tasks

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
)

type RecordingNotifierService struct {
	scheduleService *services.StateChangedEventService
	conf            *config.ExtendedConfig
}

func NewRecordingNotifierService(scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig) *RecordingNotifierService {
	return &RecordingNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
	}
}

func (srv *RecordingNotifierService) doJob() {

	Logger.Debugf("Invoked periodic RecordingNotifierService")
	ctx := context.Background()
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
