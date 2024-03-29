package tasks

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

type CleanChatsOfDeletedUserTask struct {
	*gointerlock.GoInterval
}

func CleanChatsOfDeletedUserScheduler(
	redisConnector *redisV8.Client,
	service *CleanChatsOfDeletedUserService,
) *CleanChatsOfDeletedUserTask {
	var interv = viper.GetDuration("schedulers.cleanChatsOfDeletedUserTask.interval")
	logger.Logger.Infof("Created CleanChatsOfDeletedUserScheduler with interval %v", interv)
	return &CleanChatsOfDeletedUserTask{&gointerlock.GoInterval{
		Name:           "cleanChatsOfDeletedUserTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}

type CleanChatsOfDeletedUserService struct {
	chatClient         *client.RestClient
	tracer             trace.Tracer
	dbR 			   *db.DB
}

func (srv *CleanChatsOfDeletedUserService) doJob() {
	ctx, span := srv.tracer.Start(context.Background(), "scheduler.cleanChatsOfDeletedUser")
	defer span.End()
	srv.processChats(ctx)
}

func (srv *CleanChatsOfDeletedUserService) processChats(c context.Context) {
	logger.Logger.Infof("Starting cleaning chats of deleted user job")

	// batch by chats // ... order by id
	var hasMoreChats = true
	for chatPage := 0 ; hasMoreChats; chatPage++ {
		err := db.Transact(srv.dbR, func(tx *db.Tx) error {
			chatIds, err := tx.GetChatIds(utils.DefaultSize, utils.GetOffset(chatPage, utils.DefaultSize))
			if err != nil {
				return err
			}
			hasMoreChats = len(chatIds) == utils.DefaultSize
			for _, chatId := range chatIds {

				err := tx.IterateOverAllParticipantIds(chatId, func(participantIds []int64) error {
					if err != nil {
						logger.GetLogEntry(c).Errorf("Got error getting participantIds %v", err)
						return nil
					}

					existResponse, err := srv.chatClient.CheckAreUsersExists(participantIds, c)
					if err != nil {
						logger.GetLogEntry(c).Errorf("Got error getting existResponse %v", err)
						return nil
					}

					for _, userExists := range *existResponse {
						// if user not exists
						// indeed it's kinda duplication of all the constraints on delete cascade in chat
						if !userExists.Exists {
							// remove message_read
							err = tx.DeleteMessageRead(userExists.UserId, chatId)
							if err != nil {
								logger.GetLogEntry(c).Errorf("Got error DeleteMessageRead %v", err)
								continue
							}
							// remove from chat_participants
							err = tx.DeleteParticipant(userExists.UserId, chatId)
							if err != nil {
								logger.GetLogEntry(c).Errorf("Got error DeleteMessageRead %v", err)
								continue
							}
							err = tx.DeleteChatsPinned(userExists.UserId)
							if err != nil {
								logger.GetLogEntry(c).Errorf("Got error DeleteChatsPinned %v", err)
								continue
							}
						}
					}
					return nil
				})

				// if chat has 0 participants - then remove chat
				hasParticipants, err := tx.HasParticipants(chatId)
				if err != nil {
					logger.GetLogEntry(c).Errorf("Got error HasParticipants %v", err)
					continue
				}
				if !hasParticipants {
					logger.GetLogEntry(c).Infof("Deleteing chat %v because it does not have participants", chatId)
					err = tx.DeleteChat(chatId)
					if err != nil {
						logger.GetLogEntry(c).Errorf("Got error DeleteChat %v", err)
						continue
					}
				}
			}
			return nil
		})
		if err != nil {
			logger.GetLogEntry(c).Errorf("Got error in the portion, chatPage %v, error %v", chatPage, err)
		}

	}

	logger.GetLogEntry(c).Infof("End of cleaning chats of deleted user job")
}


func NewCleanChatsOfDeletedUserService(chatClient *client.RestClient, dbR *db.DB) *CleanChatsOfDeletedUserService {
	trcr := otel.Tracer("scheduler/clean-chats-of-deleted-user")
	return &CleanChatsOfDeletedUserService{
		chatClient:         chatClient,
		tracer:             trcr,
		dbR: 				dbR,
	}
}
