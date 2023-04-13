package redis

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"nkonev.name/video/client"
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
	chatClient              *client.RestClient
}

func NewChatDialerService(scheduleService *services.DialRedisRepository, conf *config.ExtendedConfig, rabbitMqInvitePublisher *producer.RabbitInvitePublisher, dialStatusPublisher *producer.RabbitDialStatusPublisher, chatClient *client.RestClient) *ChatDialerService {
	return &ChatDialerService{
		redisService:            scheduleService,
		conf:                    conf,
		rabbitMqInvitePublisher: rabbitMqInvitePublisher,
		dialStatusPublisher:     dialStatusPublisher,
		chatClient:              chatClient,
	}
}

func (srv *ChatDialerService) doJob() {

	if srv.conf.VideoCallUsersCountNotificationPeriod == 0 {
		Logger.Debugf("Scheduler in ChatDialerService is disabled")
		return
	}

	Logger.Debugf("Invoked periodic ChatDialer")
	ctx := context.Background()
	chats, err := srv.redisService.GetDialChats(ctx)
	if err != nil {
		Logger.Errorf("Error %v", err)
		return
	}

	for _, chatId := range chats {
		srv.makeDial(ctx, chatId)
		srv.checkAndRemoveRedundants(ctx, chatId)
	}

	Logger.Debugf("End of ChatNotifier")
}

func (srv *ChatDialerService) makeDial(ctx context.Context, chatId int64) {
	behalfUserId, err := srv.redisService.GetDialMetadata(ctx, chatId)
	if err != nil {
		Logger.Warnf("Error %v", err)
		return
	}
	userIdsToDial, err := srv.redisService.GetUsersToDial(ctx, chatId)
	if err != nil {
		Logger.Warnf("Error %v", err)
		return
	}

	Logger.Infof("Calling userIds %v from chat %v", userIdsToDial, chatId)

	inviteNames, err := srv.chatClient.GetChatNameForInvite(chatId, behalfUserId, userIdsToDial, ctx)
	if err != nil {
		Logger.Error(err, "Failed during getting chat invite names")
		return
	}

	for _, chatInviteName := range inviteNames {
		invitation := dto.VideoCallInvitation{
			ChatId:   chatId,
			ChatName: chatInviteName.Name,
		}

		err = srv.rabbitMqInvitePublisher.Publish(&invitation, chatInviteName.UserId)
		if err != nil {
			Logger.Error(err, "Failed during marshal VideoInviteDto")
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
		Logger.Error(err, "Failed during marshal VideoIsInvitingDto")
		return
	}
}

func (srv *ChatDialerService) SendDialStatusChanged(ctx context.Context, behalfUserId int64, chatId int64) {
	userIdsToDial, err := srv.redisService.GetUsersToDial(ctx, chatId)
	if err != nil {
		Logger.Warnf("Error %v", err)
		return
	}

	var videoIsInvitingDto = dto.VideoIsInvitingDto{
		ChatId:       chatId,
		UserIds:      userIdsToDial,
		Status:       true,
		BehalfUserId: behalfUserId,
	}
	err = srv.dialStatusPublisher.Publish(&videoIsInvitingDto)
	if err != nil {
		Logger.Error(err, "Failed during marshal VideoIsInvitingDto")
		return
	}
}

func (srv *ChatDialerService) checkAndRemoveRedundants(ctx context.Context, chatId int64) {
	userIdsToDial, err := srv.redisService.GetUsersToDial(ctx, chatId)
	if err != nil {
		Logger.Warnf("Error %v", err)
		return
	}
	participantBelongToChat, err := srv.chatClient.DoesParticipantBelongToChat(chatId, userIdsToDial, ctx)
	if err != nil {
		Logger.Warnf("Error %v", err)
		return
	}

	for _, userBelongsInfo := range participantBelongToChat {
		if !userBelongsInfo.Belongs {
			err := srv.redisService.RemoveFromDialList(ctx, userBelongsInfo.UserId, chatId)
			if err != nil {
				Logger.Warnf("Error %v", err)
			}
		}
	}
}

type ChatDialerTask struct {
	*gointerlock.GoInterval
}

func ChatDialerScheduler(
	redisConnector *redisV8.Client,
	service *ChatDialerService,
	conf *config.ExtendedConfig,
) *ChatDialerTask {
	var interv = conf.VideoCallUsersCountNotificationPeriod
	Logger.Infof("Created chats dialer with interval %v", interv)
	return &ChatDialerTask{&gointerlock.GoInterval{
		Name:           "chatDialer",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
