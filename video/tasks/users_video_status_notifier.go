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

type UsersVideoStatusNotifierService struct {
	scheduleService *services.StateChangedEventService
	conf            *config.ExtendedConfig
}

func NewUsersVideoStatusNotifierService(scheduleService *services.StateChangedEventService, conf *config.ExtendedConfig) *UsersVideoStatusNotifierService {
	return &UsersVideoStatusNotifierService{
		scheduleService: scheduleService,
		conf:            conf,
	}
}

func (srv *UsersVideoStatusNotifierService) doJob() {

	Logger.Debugf("Invoked periodic UsersVideoStatusNotifier")
	ctx := context.Background()
	srv.scheduleService.NotifyAllChatsAboutUsersVideoStatus(ctx)

	Logger.Debugf("End of UsersVideoStatusNotifier")
}

type UsersVideoStatusNotifierTask struct {
	*gointerlock.GoInterval
}

func UsersVideoStatusNotifierScheduler(
	redisConnector *redisV8.Client,
	service *UsersVideoStatusNotifierService,
	conf *config.ExtendedConfig,
) *UsersVideoStatusNotifierTask {
	var interv = viper.GetDuration("schedulers.usersVideoStatusNotifierTask.notificationPeriod")
	Logger.Infof("Created users video status periodic notificator with interval %v", interv)
	return &UsersVideoStatusNotifierTask{&gointerlock.GoInterval{
		Name:           "usersVideoStatusNotifierTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
