package redis

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"nkonev.name/video/config"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/services"
)

type ChatDialerService struct {
	redisService            *services.DialRedisRepository
	conf                    *config.ExtendedConfig
	rabbitMqInvitePublisher *producer.RabbitInvitePublisher
	dialStatusPublisher     *producer.RabbitDialStatusPublisher
}

func NewChatDialerService(scheduleService *services.DialRedisRepository, conf *config.ExtendedConfig, rabbitMqInvitePublisher *producer.RabbitInvitePublisher, dialStatusPublisher *producer.RabbitDialStatusPublisher) *ChatDialerService {
	return &ChatDialerService{
		redisService:            scheduleService,
		conf:                    conf,
		rabbitMqInvitePublisher: rabbitMqInvitePublisher,
		dialStatusPublisher:     dialStatusPublisher,
	}
}

func (srv *ChatDialerService) doJob() {

	if srv.conf.SyncNotificationPeriod == 0 {
		Logger.Info("Scheduler in ChatDialerService is disabled")
		return
	}

	Logger.Info("Invoked periodic ChatDialer")
	ctx := context.Background()
	chats, err := srv.redisService.GetDialChats(ctx)
	if err != nil {
		Logger.Errorf("Error %v", err)
		return
	}

	for _, chatId := range chats {
		behalfUserId, behalfUserLogin, err := srv.redisService.GerDialMetadata(ctx, chatId)
		if err != nil {
			Logger.Warnf("Error %v", err)
			continue
		}
		userIdsToDial, err := srv.redisService.GetUsersToDial(ctx, chatId)
		if err != nil {
			Logger.Warnf("Error %v", err)
			continue
		}

		for _, userId := range userIdsToDial {
			inviteDto := dto.VideoInviteDto{
				ChatId:       chatId,
				UserId:       userId,
				BehalfUserId: behalfUserId,
				BehalfLogin:  behalfUserLogin,
			}

			Logger.Infof("Calling userId %v from chat %v", userId, chatId)
			err = srv.rabbitMqInvitePublisher.Publish(&inviteDto)
			if err != nil {
				Logger.Error(err, "Failed during marshal chatNotifyDto")
				continue
			}
		}

		// send state changes
		var videoIsInvitingDto = dto.VideoIsInvitingDto{
			ChatId:       chatId,
			UserIds:      userIdsToDial,
			Status:       true,
			BehalfUserId: behalfUserId,
		}
		err = srv.dialStatusPublisher.Publish(&videoIsInvitingDto)
		if err != nil {
			Logger.Error(err, "Failed during marshal chatNotifyDto")
			continue
		}
	}

	Logger.Infof("End of ChatNotifier")
}

type ChatDialerTask struct {
	*gointerlock.GoInterval
}

func ChatDialerScheduler(
	redisConnector *redisV8.Client,
	service *ChatDialerService,
	conf *config.ExtendedConfig,
) *ChatDialerTask {
	var interv = conf.SyncNotificationPeriod
	Logger.Infof("Created chats dialer with interval %v", interv)
	return &ChatDialerTask{&gointerlock.GoInterval{
		Name:           "chatDialer",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
