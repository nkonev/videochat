package tasks

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
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
	tracer             trace.Tracer
	chatInvitationService *services.ChatInvitationService
}

func NewChatDialerService(scheduleService *services.DialRedisRepository, conf *config.ExtendedConfig, rabbitMqInvitePublisher *producer.RabbitInvitePublisher, dialStatusPublisher *producer.RabbitDialStatusPublisher, chatClient *client.RestClient, chatInvitationService *services.ChatInvitationService) *ChatDialerService {
	trcr := otel.Tracer("scheduler/chat-dialer")
	return &ChatDialerService{
		redisService:            scheduleService,
		conf:                    conf,
		rabbitMqInvitePublisher: rabbitMqInvitePublisher,
		dialStatusPublisher:     dialStatusPublisher,
		chatClient:              chatClient,
		tracer:             trcr,
		chatInvitationService: chatInvitationService,
	}
}

func (srv *ChatDialerService) doJob() {

	ctx, span := srv.tracer.Start(context.Background(), "scheduler.ChatDialer")
	defer span.End()

	Logger.Debugf("Invoked periodic ChatDialer")

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
	ownerId, err := srv.redisService.GetDialMetadata(ctx, chatId)
	if err != nil {
		GetLogEntry(ctx).Warnf("Error %v", err)
		return
	}
	userIdsToDial, err := srv.redisService.GetUsersToDial(ctx, chatId)
	if err != nil {
		GetLogEntry(ctx).Warnf("Error %v", err)
		return
	}

	GetLogEntry(ctx).Infof("Calling userIds %v from chat %v", userIdsToDial, chatId)

	srv.chatInvitationService.SendInvitations(ctx, chatId, ownerId, userIdsToDial)

	// send state changes to owner (ownerId) of call
	err = srv.dialStatusPublisher.Publish(chatId, userIdsToDial, true, ownerId)
	if err != nil {
		GetLogEntry(ctx).Error(err, "Failed during marshal VideoIsInvitingDto")
		return
	}
}

// removes users from dial who were removed from chat
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
			// remove call users who were removed as participants from the chat
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
	var interv = viper.GetDuration("schedulers.chatDialerTask.dialPeriod")
	Logger.Infof("Created chats dialer with interval %v", interv)
	return &ChatDialerTask{&gointerlock.GoInterval{
		Name:           "chatDialerTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
