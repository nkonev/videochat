package redis

import (
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
	"nkonev.name/video/services"
)

type ChatNotifierService struct {
	scheduleService *services.StateChangedNotificationService
	conf            *config.ExtendedConfig
}

func NewChatNotifierService(scheduleService *services.StateChangedNotificationService, conf *config.ExtendedConfig) *ChatNotifierService {
	return &ChatNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
	}
}

func (srv *ChatNotifierService) doJob() {

	if srv.conf.SyncNotificationPeriod == 0 {
		Logger.Info("Scheduler in ChatNotifierService is disabled")
		return
	}

	Logger.Info("Invoked periodic ChatNotifier")
	srv.scheduleService.NotifyAllChats()

	Logger.Infof("End of ChatNotifier")
}

type ChatNotifierTask struct {
	*gointerlock.GoInterval
}

func ChatNotifierScheduler(
	redisConnector *redisV8.Client,
	service *ChatNotifierService,
	conf *config.ExtendedConfig,
) *ChatNotifierTask {
	var interv = conf.SyncNotificationPeriod
	Logger.Infof("Created chats periodic notificator with interval %v", interv)
	return &ChatNotifierTask{&gointerlock.GoInterval{
		Name:           "chatPeriodicNotifier",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
