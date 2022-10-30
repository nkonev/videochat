package redis

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
)

type VideoCallUsersCountNotifierService struct {
	scheduleService *services.StateChangedNotificationService
	conf            *config.ExtendedConfig
}

func NewVideoCallUsersCountNotifierService(scheduleService *services.StateChangedNotificationService, conf *config.ExtendedConfig) *VideoCallUsersCountNotifierService {
	return &VideoCallUsersCountNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
	}
}

func (srv *VideoCallUsersCountNotifierService) doJob() {

	if srv.conf.VideoCallUsersCountNotificationPeriod == 0 {
		Logger.Debugf("Scheduler in VideoCallUsersCountNotifierService is disabled")
		return
	}

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
	var interv = conf.VideoCallUsersCountNotificationPeriod
	Logger.Infof("Created video call users count periodic notificator with interval %v", interv)
	return &VideoCallUsersCountNotifierTask{&gointerlock.GoInterval{
		Name:           "videoCallUsersCountPeriodicNotifier",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
