package redis

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
)

type VideoCallUsersCountNotifierService struct {
	scheduleService *services.StateChangedEventService
	conf            *config.ExtendedConfig
}

func NewVideoCallUsersCountNotifierService(scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig) *VideoCallUsersCountNotifierService {
	return &VideoCallUsersCountNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
	}
}

func (srv *VideoCallUsersCountNotifierService) doJob() {

	Logger.Debugf("Invoked periodic ChatNotifier")
	ctx := context.Background()
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
