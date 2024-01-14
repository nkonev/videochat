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
	"time"
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

	var statuses = srv.GetStatuses(ctx, userIdsToDial)

	GetLogEntry(ctx).Infof("Calling userIds %v from chat %v", userIdsToDial, chatId)

	// send invitations to callees
	srv.chatInvitationService.SendInvitationsWithStatuses(ctx, chatId, ownerId, statuses)

	// send state changes to owner (ownerId) of call
	err = srv.dialStatusPublisher.Publish(chatId, statuses, ownerId)
	if err != nil {
		GetLogEntry(ctx).Error(err, "Failed during marshal VideoIsInvitingDto")
	}

	// cleanNotNeededAnymoreDialRedisData
	srv.cleanNotNeededAnymoreDialRedisData(ctx, chatId, ownerId, userIdsToDial)
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

func (srv *ChatDialerService) GetStatuses(ctx context.Context, userIds []int64) map[int64]string {
	var ret = map[int64]string{}
	for _, userId := range userIds {
		status, err := srv.redisService.GetUserCallStatus(ctx, userId)
		if err != nil {
			GetLogEntry(ctx).Error("An error occured during getting the status for user %", userId)
			continue
		}
		if status != services.CallStatusNotFound {
			ret[userId] = status
		}
	}
	return ret
}

func (srv *ChatDialerService) cleanNotNeededAnymoreDialRedisData(ctx context.Context, chatId int64, ownerId int64, userIdsToDial []int64) {
	for _, userId := range userIdsToDial {
		userCallState, _, userCallMarkedForRemoveAt, _, err := srv.redisService.GetUserCallState(ctx, userId)
		if err != nil {
			GetLogEntry(ctx).Errorf("Unable to get user call state %v", err)
			continue
		}
		if userCallState == services.CallStatusNotFound { // it can be un case when we override an user's temporary status
			err := srv.redisService.RemoveFromDialList(ctx, userId, chatId)
			if err != nil {
				GetLogEntry(ctx).Errorf("Unable invoke RemoveFromDialList, user %v, error %v", userId, err)
				continue
			}
		} else if services.IsTemporary(userCallState) {
			if userCallMarkedForRemoveAt != services.UserCallMarkedForRemoveAtNotSet &&
			  time.Now().Sub(time.UnixMilli(userCallMarkedForRemoveAt)) > viper.GetDuration("schedulers.chatDialerTask.removeTemporaryUserCallStatusAfter") {

				GetLogEntry(ctx).Infof("Removing temporary UserCallStatus of user %v", userId)
				err := srv.redisService.RemoveFromDialList(ctx, userId, chatId)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable invoke RemoveFromDialList, user %v, error %v", userId, err)
					continue
				}
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
