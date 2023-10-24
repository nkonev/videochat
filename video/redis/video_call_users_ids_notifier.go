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

type VideoCallUsersIdsNotifierService struct {
	scheduleService *services.StateChangedEventService
	conf            *config.ExtendedConfig
}

func NewVideoCallUsersIdsNotifierService(scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig) *VideoCallUsersIdsNotifierService {
	return &VideoCallUsersIdsNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
	}
}

func (srv *VideoCallUsersIdsNotifierService) doJob() {

	Logger.Debugf("Invoked periodic ChatNotifier")
	ctx := context.Background()
	srv.scheduleService.NotifyAllChatsAboutVideoCallUsersIds(ctx)

	Logger.Debugf("End of ChatNotifier")
}

type VideoCallUsersIdsNotifierTask struct {
	*gointerlock.GoInterval
}

func VideoCallUsersIdsNotifierScheduler(
	redisConnector *redisV8.Client,
	service *VideoCallUsersIdsNotifierService,
	conf *config.ExtendedConfig,
) *VideoCallUsersIdsNotifierTask {
	var interv = viper.GetDuration("schedulers.videoCallUsersIdsNotifierTask.notificationPeriod")
	Logger.Infof("Created video call users count periodic notificator with interval %v", interv)
	return &VideoCallUsersIdsNotifierTask{&gointerlock.GoInterval{
		Name:           "videoCallUsersIdsNotifierTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
