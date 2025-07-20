package tasks

import (
	"context"
	"github.com/nkonev/dcron"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
)

type CleanChatsOfDeletedUserTask struct {
	dcron.Job
}

func CleanChatsOfDeletedUserScheduler(
	lgr *logger.Logger,
	service *CleanChatsOfDeletedUserService,
) *CleanChatsOfDeletedUserTask {
	const key = "cleanChatsOfDeletedUserTask"
	var str = viper.GetString("schedulers." + key + ".cron")
	lgr.Infof("Created CleanChatsOfDeletedUserScheduler with cron %v", str)

	job := dcron.NewJob(key, str, func(ctx context.Context) error {
		service.doJob(ctx)
		return nil
	}, dcron.WithTracing(service.spanStarter, service.spanFinisher))

	return &CleanChatsOfDeletedUserTask{job}
}

type CleanChatsOfDeletedUserService struct {
	restClient *client.RestClient
	tracer     trace.Tracer
	dbR        *db.DB
	lgr        *logger.Logger
}

func (srv *CleanChatsOfDeletedUserService) doJob(ctx context.Context) {
	srv.processChats(ctx)
}

func (srv *CleanChatsOfDeletedUserService) processChats(c context.Context) {
	srv.lgr.WithTracing(c).Infof("Starting cleaning chats of deleted user job")

	err := db.Transact(c, srv.dbR, func(tx *db.Tx) error {
		return tx.IterateOverAllParticipantIds(c, func(participantIds []int64) error {
			existResponse, err := srv.restClient.CheckAreUsersExists(c, participantIds)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Got error getting existResponse %v", err)
				return nil
			}
			if existResponse == nil {
				srv.lgr.WithTracing(c).Errorf("Got null getting existResponse %v", err)
				return nil
			}

			for _, userExists := range *existResponse {
				if !userExists.Exists {
					// remove message_read
					srv.lgr.WithTracing(c).Infof("Deleteing message read for user %v", userExists.UserId)
					err = tx.DeleteAllMessageRead(c, userExists.UserId)
					if err != nil {
						srv.lgr.WithTracing(c).Errorf("Got error DeleteMessageRead %v", err)
					}
					// remove from chat_participants
					srv.lgr.WithTracing(c).Infof("Deleteing patricipance for user %v", userExists.UserId)
					err = tx.DeleteUserAsAParticipantFromAllChats(c, userExists.UserId)
					if err != nil {
						srv.lgr.WithTracing(c).Errorf("Got error DeleteMessageRead %v", err)
					}
					srv.lgr.WithTracing(c).Infof("Deleteing pinned chats for user %v", userExists.UserId)
					err = tx.DeleteChatsPinned(c, userExists.UserId)
					if err != nil {
						srv.lgr.WithTracing(c).Errorf("Got error DeleteChatsPinned %v", err)
					}
					srv.lgr.WithTracing(c).Infof("Deleteing notification settings for user %v", userExists.UserId)
					err = tx.DeleteAllChatParticipantNotification(c, userExists.UserId)
					if err != nil {
						srv.lgr.WithTracing(c).Errorf("Got error DeleteMessageRead %v", err)
					}
				}
			}
			return nil
		})
	})
	if err != nil {
		srv.lgr.WithTracing(c).Errorf("Got error during remove an user leftovers %v", err)
	}

	// batch by chats // ... order by id
	var hasMoreChats = true
	for chatPage := 0; hasMoreChats; chatPage++ {
		err := db.Transact(c, srv.dbR, func(tx *db.Tx) error {
			chatIds, err := tx.GetChatIds(c, utils.DefaultSize, utils.GetOffset(chatPage, utils.DefaultSize))
			if err != nil {
				return err
			}
			hasMoreChats = len(chatIds) == utils.DefaultSize

			hasParticipantsMap, err := tx.HasParticipants(c, chatIds)
			if err != nil {
				srv.lgr.WithTracing(c).Errorf("Got error HasParticipants %v", err)
				return nil
			}

			for _, chatId := range chatIds {
				// if chat has 0 participants - then remove chat
				hasParticipants := hasParticipantsMap[chatId]
				if !hasParticipants {
					srv.lgr.WithTracing(c).Infof("Deleteing chat %v because it does not have participants", chatId)
					err = tx.DeleteChat(c, chatId)
					if err != nil {
						srv.lgr.WithTracing(c).Errorf("Got error DeleteChat %v", err)
						continue
					}
				}
			}
			return nil
		})
		if err != nil {
			srv.lgr.WithTracing(c).Errorf("Got error in the portion, chatPage %v, error %v", chatPage, err)
		}

	}

	srv.lgr.WithTracing(c).Infof("End of cleaning chats of deleted user job")
}

func (srv *CleanChatsOfDeletedUserService) spanStarter(ctx context.Context) (context.Context, any) {
	return srv.tracer.Start(ctx, "scheduler.cleanChatsOfDeletedUser")
}

func (srv *CleanChatsOfDeletedUserService) spanFinisher(ctx context.Context, span any) {
	span.(trace.Span).End()
}

func NewCleanChatsOfDeletedUserService(lgr *logger.Logger, chatClient *client.RestClient, dbR *db.DB) *CleanChatsOfDeletedUserService {
	trcr := otel.Tracer("scheduler/clean-chats-of-deleted-user")
	return &CleanChatsOfDeletedUserService{
		restClient: chatClient,
		tracer:     trcr,
		dbR:        dbR,
		lgr:        lgr,
	}
}
