package tasks

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/livekit/protocol/livekit"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	"nkonev.name/video/db"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
)

type SynchronizeWithLivekitService struct {
	database                  *db.DB
	userService               *services.UserService
	tracer                    trace.Tracer
	livekitRoomClient         client.LivekitRoomClient
	restClient                *client.RestClient
	rabbitUserIdsPublisher    *producer.RabbitUserIdsPublisher
	rabbitUserInvitePublisher *producer.RabbitInvitePublisher
}

func NewSynchronizeWithLivekitService(
	database *db.DB,
	userService *services.UserService,
	livekitRoomClient client.LivekitRoomClient,
	restClient *client.RestClient,
	rabbitUserIdsPublisher *producer.RabbitUserIdsPublisher,
	rabbitUserInvitePublisher *producer.RabbitInvitePublisher,
) *SynchronizeWithLivekitService {
	trcr := otel.Tracer("scheduler/synchronize-with-livekit")
	return &SynchronizeWithLivekitService{
		database:                  database,
		userService:               userService,
		tracer:                    trcr,
		livekitRoomClient:         livekitRoomClient,
		restClient:                restClient,
		rabbitUserIdsPublisher:    rabbitUserIdsPublisher,
		rabbitUserInvitePublisher: rabbitUserInvitePublisher,
	}
}

func (srv *SynchronizeWithLivekitService) DoJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.SynchronizeWithLivekit")
	defer span.End()

	GetLogEntry(ctx).Debugf("Invoked periodic SynchronizeWithLivekit")

	srv.cleanOrphans(ctx)

	srv.createParticipants(ctx)
}

func (srv *SynchronizeWithLivekitService) cleanOrphans(ctx context.Context) {
	offset := int64(0)
	hasMoreElements := true
	for hasMoreElements {
		err := db.Transact(srv.database, func(tx *db.Tx) error {
			// here we use order by owner_id
			batchUserStates, err := tx.GetAllUserStates(utils.DefaultSize, offset)
			if err != nil {
				GetLogEntry(ctx).Errorf("error during reading user states %v", err)
				return err
			}
			srv.processBatch(ctx, tx, batchUserStates)

			hasMoreElements = len(batchUserStates) == utils.DefaultSize
			offset += utils.DefaultSize

			return nil
		})

		if err != nil {
			GetLogEntry(ctx).Errorf("error during processing: %v", err)
			continue
		}
	}
}

func (srv *SynchronizeWithLivekitService) processBatch(ctx context.Context, tx *db.Tx, batchUserStates []dto.UserCallState) {

	// move orphaned users in "inCall" status to "cancelling"
	for _, st := range batchUserStates {
		chatId := st.ChatId
		userCallStateId := dto.UserCallStateId{
			TokenId: st.TokenId,
			UserId:  st.UserId,
		}
		// consider only users, hanged in "inCall" state in redis and not presented in livekit
		// you need to start reading from 1.
		if st.Status == db.CallStatusInCall {
			// 2. removing
			if st.MarkedForOrphanRemoveAttempt >= viper.GetInt("schedulers.synchronizeWithLivekitTask.orphanUserIteration") {
				GetLogEntry(ctx).Warnf("Removing owned call by user tokenId %v, userId %v because attempts were exhausted", st.TokenId, st.UserId)
				// case 2.a user is owner of the call
				// soft remove owned (callee, invitee) by user
				invitedByMe, err := tx.GetBeingInvitedByOwnerId(userCallStateId, chatId)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to find owned by user tokenId %v, userId %v, chatId %v, error: %v", st.TokenId, st.UserId, chatId, err)
				}
				for _, invitee := range invitedByMe {
					err = tx.SetRemoving(dto.UserCallStateId{invitee.TokenId, invitee.UserId}, db.CallStatusRemoving)
					if err != nil {
						GetLogEntry(ctx).Errorf("Unable to move invitee to remoning status owned by user tokenId %v, userId %v, chatId %v, error: %v", st.TokenId, st.UserId, chatId, err)
					}
				}
				// case 2.b user is just user
				// soft remove the user
				err = tx.SetRemoving(userCallStateId, db.CallStatusRemoving)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to move invitee to remoning status owned by user tokenId %v, userId %v, chatId %v, error: %v", st.TokenId, st.UserId, chatId, err)
				}

				err = srv.rabbitUserIdsPublisher.Publish(ctx, &dto.VideoCallUsersCallStatusChangedDto{Users: []dto.VideoCallUserCallStatusChangedDto{
					{
						UserId:    st.UserId,
						IsInVideo: false,
					},
				}})
				if err != nil {
					GetLogEntry(ctx).Errorf("Error during notifying about user is in video, userId=%v, chatId=%v, error=%v", st.UserId, chatId, err)
				}

				continue // because we don't need increment an attempt
			}

			// 1. changing attempt number
			videoParticipants, err := srv.userService.GetVideoParticipants(ctx, chatId)
			if err != nil {
				GetLogEntry(ctx).Errorf("Unable to get video participants of %v", chatId)
				continue
			}
			if !srv.Contains(videoParticipants, userCallStateId) {
				newAttempt := st.MarkedForOrphanRemoveAttempt + 1
				GetLogEntry(ctx).Infof("Setting attempt %v on userCallState %v of user tokenId %v, userId %v because they aren't among video room participants", newAttempt, st.Status, st.TokenId, st.UserId)
				err = tx.SetMarkedForOrphanRemoveAttempt(userCallStateId, newAttempt)
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to set user markedForChangeStatusAttempt user tokenId %v, userId %v", st.TokenId, st.UserId)
					continue
				}
			} else {
				if st.MarkedForOrphanRemoveAttempt >= 1 {
					GetLogEntry(ctx).Infof("Resetting attempt on userCallState %v of user tokenId %v, userId %v because they appeared among video room participants", st.Status, userCallStateId.TokenId, userCallStateId.UserId)

					err = tx.SetMarkedForOrphanRemoveAttempt(userCallStateId, db.UserCallMarkedForOrphanRemoveAttemptNotSet)
					if err != nil {
						GetLogEntry(ctx).Errorf("Unable to set user markedForChangeStatusAttempt user tokenId %v, userId %v", st.TokenId, st.UserId)
						continue
					}
				}
			}

		} // else branch not needed, because they removed from chat_dialer task's cleanNotNeededAnymoreDialData()
	}
}

func (srv *SynchronizeWithLivekitService) createParticipants(ctx context.Context) {
	listRoomReq := &livekit.ListRoomsRequest{}
	rooms, err := srv.livekitRoomClient.ListRooms(ctx, listRoomReq)
	if err != nil {
		GetLogEntry(ctx).Error(err, "error during reading rooms %v", err)
		return
	}

	for _, room := range rooms.Rooms {
		chatId, err := utils.GetRoomIdFromName(room.Name)
		if err != nil {
			GetLogEntry(ctx).Errorf("got error during getting chat id from roomName %v %v", room.Name, err)
			continue
		}

		videoParticipants, err := srv.userService.GetVideoParticipants(ctx, chatId)
		if err != nil {
			GetLogEntry(ctx).Errorf("got error during getting videoParticipants from roomName %v %v", room.Name, err)
			continue
		}

		// if no such users
		for _, videoParticipant := range videoParticipants {
			err = db.Transact(srv.database, func(tx *db.Tx) error {
				userState, err := tx.Get(dto.UserCallStateId{
					TokenId: videoParticipant.TokenId,
					UserId:  videoParticipant.UserId,
				})
				if err != nil {
					GetLogEntry(ctx).Errorf("Unable to get user call state %v", err)
					return err
				}

				// if there is no status in redis, but we have it in livekit - then create
				if userState.Status == db.CallStatusNotFound {
					GetLogEntry(ctx).Warnf("Populating user with tokenId %v userId %v from livekit to redis in chat %v", videoParticipant.TokenId, videoParticipant.UserId, chatId)

					chatInfo, err := srv.restClient.GetBasicChatInfo(ctx, chatId, videoParticipant.UserId)
					if err != nil {
						GetLogEntry(ctx).Errorf("Unable to GetBasicChatInfo %v", err)
						return err
					}

					err = tx.AddAsEntered(videoParticipant.TokenId, videoParticipant.UserId, chatId, chatInfo.TetATet)
					if err != nil {
						GetLogEntry(ctx).Errorf("Unable to AddToDialList %v", err)
						return err
					}
				}
				return nil
			})
			if err != nil {
				GetLogEntry(ctx).Errorf("Error: %v", err)
				continue
			}
		}
	}
}

func (srv *SynchronizeWithLivekitService) Contains(participants []dto.UserCallStateId, id dto.UserCallStateId) bool {
	for _, p := range participants {
		if p == id {
			return true
		}
	}
	return false
}

type SynchronizeWithLivekitTask struct {
	*gointerlock.GoInterval
}

func SynchronizeWithLivekitSheduler(
	redisConnector *redisV8.Client,
	service *SynchronizeWithLivekitService,
	conf *config.ExtendedConfig,
) *SynchronizeWithLivekitTask {
	var interv = viper.GetDuration("schedulers.synchronizeWithLivekitTask.period")
	Logger.Infof("Synchronize with livekit task with interval %v", interv)
	return &SynchronizeWithLivekitTask{&gointerlock.GoInterval{
		Name:           "synchronizeWithLivekitTask",
		Interval:       interv,
		Arg:            service.DoJob,
		RedisConnector: redisConnector,
	}}
}
