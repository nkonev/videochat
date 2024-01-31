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
	stateChangedEventService *services.StateChangedEventService
}

func NewChatDialerService(
	scheduleService *services.DialRedisRepository,
	conf *config.ExtendedConfig,
	rabbitMqInvitePublisher *producer.RabbitInvitePublisher,
	dialStatusPublisher *producer.RabbitDialStatusPublisher,
	chatClient *client.RestClient,
	stateChangedEventService *services.StateChangedEventService,
) *ChatDialerService {
	trcr := otel.Tracer("scheduler/chat-dialer")
	return &ChatDialerService{
		redisService:            scheduleService,
		conf:                    conf,
		rabbitMqInvitePublisher: rabbitMqInvitePublisher,
		dialStatusPublisher:     dialStatusPublisher,
		chatClient:              chatClient,
		tracer:             trcr,
		stateChangedEventService: stateChangedEventService,
	}
}

func (srv *ChatDialerService) doJob() {

	ctx, span := srv.tracer.Start(context.Background(), "scheduler.ChatDialer")
	defer span.End()

	Logger.Debugf("Invoked periodic ChatDialer")

	usersOwners, err := srv.redisService.GetUsersOwesCalls(ctx)
	if err != nil {
		Logger.Errorf("Error %v", err)
		return
	}

	for _, ownerId := range usersOwners {
		srv.makeDial(ctx, ownerId)
	}

	Logger.Debugf("End of ChatNotifier")
}

func (srv *ChatDialerService) makeDial(ctx context.Context, ownerId int64) {
	userIdsToDial, err := srv.redisService.GetUsersOfOwnersDial(ctx, ownerId)
	if err != nil {
		GetLogEntry(ctx).Warnf("Error %v", err)
		return
	}

	for _, userId := range userIdsToDial {
		status, chatId, userCallMarkedForRemoveAt, _, _, err := srv.redisService.GetUserCallState(ctx, userId)
		if err != nil {
			GetLogEntry(ctx).Errorf("An error occured during getting the status for user %v: %v", userId, err)
			continue
		}
		// cleanNotNeededAnymoreDialRedisData - should be before status == services.CallStatusNotFound exit
		srv.cleanNotNeededAnymoreDialRedisData(ctx, chatId, ownerId, userId, status, userCallMarkedForRemoveAt, userIdsToDial)

		if status == services.CallStatusNotFound {
			GetLogEntry(ctx).Warnf("Call status isn't found for user %v", userId)
			continue
		}

		GetLogEntry(ctx).Infof("Sending userCallStatus for userIds %v from ownerId %v", userIdsToDial, ownerId)
		// send invitations to callees
		srv.stateChangedEventService.SendInvitationsWithStatuses(ctx, chatId, ownerId, map[int64]string{userId: status})
		// send state changes to owner (ownerId) of call
		srv.dialStatusPublisher.Publish(chatId, map[int64]string{userId: status}, ownerId)
	}

}

// chatId can be NoChat
func (srv *ChatDialerService) cleanNotNeededAnymoreDialRedisData(
	ctx context.Context,
	chatId int64,
	ownerId int64,
	userId int64,
	userCallState string,
	userCallMarkedForRemoveAt int64,
	userIdsOfDial []int64,
) {
	if userCallState == services.CallStatusNotFound { // shouldn't happen
		GetLogEntry(ctx).Warnf("Going to remove excess data for user %v, chat %v", userId, chatId)
		err := srv.redisService.RemoveFromDialList(ctx, userId, true, ownerId)
		if err != nil {
			GetLogEntry(ctx).Errorf("Unable invoke RemoveFromDialList, user %v, error %v", userId, err)
			return
		}
	} else if services.IsTemporary(userCallState) {
		if userCallMarkedForRemoveAt != services.UserCallMarkedForRemoveAtNotSet &&
			time.Now().Sub(time.UnixMilli(userCallMarkedForRemoveAt)) > viper.GetDuration("schedulers.chatDialerTask.removeTemporaryUserCallStatusAfter") {

			// case: tet-a-tet
			// user 1 starts video and invites user 2
			// then user 1 exits
			// if we don't do this - we will have dangling dials_of_user:1
			// delegate ownership to another user
			if ownerId == userId {
				err := srv.redisService.TransferOwnership(ctx, userIdsOfDial, ownerId, chatId)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable invoke TransferOwnership, user %v, chat %v, error %v", userId, chatId, err)
					return
				}
			}

			GetLogEntry(ctx).Infof("Removing temporary userCallStatus of user %v, chat %v", userId, chatId)
			err := srv.redisService.RemoveFromDialList(ctx, userId, true, ownerId)
			if err != nil {
				GetLogEntry(ctx).Errorf("Unable invoke RemoveFromDialList, user %v, chat %v, error %v", userId, chatId, err)
				return
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
