package tasks

import (
	"context"
	"nkonev.name/chat/client"
	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"

	"github.com/nkonev/dcron"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const CleanDeletedUserDataSchedulerKey = "cleanDeletedUserDataTask"

type CleanDeletedUserDataTask struct {
	dcron.Job
}

func CleanDeletedUserDataScheduler(
	lgr *logger.LoggerWrapper,
	service *CleanDeletedUserDataService,
	cfg *config.AppConfig,
) *CleanDeletedUserDataTask {
	var str = cfg.Schedulers.CleanDeletedUsersDataTask.Cron
	lgr.Info("Created CleanDeletedUserDataScheduler with cron", "cron", str, dcron.SlogKeyTaskName, CleanDeletedUserDataSchedulerKey)

	job := dcron.NewJob(CleanDeletedUserDataSchedulerKey, str, func(ctx context.Context) error {
		service.DoJob(ctx)
		return nil
	},
		dcron.WithTracing(service.spanStarter, service.spanFinisher),
		dcron.WithJobSettings(cfg.Schedulers.CleanDeletedUsersDataTask.Expiration),
	)

	return &CleanDeletedUserDataTask{job}
}

type CleanDeletedUserDataService struct {
	restClient client.AaaRestClient
	tracer     trace.Tracer
	dbR        *db.DB
	lgr        *logger.LoggerWrapper
	eventBus   *cqrs.KafkaProducer
	co         *cqrs.CommonProjection
}

func (srv *CleanDeletedUserDataService) DoJob(ctx context.Context) {
	srv.processChats(ctx)
}

func (srv *CleanDeletedUserDataService) processChats(c context.Context) {
	srv.lgr.InfoContext(c, "Starting cleaning deleted users data job")

	errOuter := srv.co.IterateOverAllParticipants(c, srv.dbR, func(chatParticipants []dto.ChatParticipant) error {
		userIdMap := map[int64]struct{}{}
		for _, cp := range chatParticipants {
			userIdMap[cp.UserId] = struct{}{}
		}

		existResponse, err := srv.restClient.CheckAreUsersExists(c, utils.SetMapIdStructToSlice(userIdMap))
		if err != nil {
			srv.lgr.ErrorContext(c, "Got error getting existResponse", logger.AttributeError, err)
			return nil
		}
		if existResponse == nil {
			srv.lgr.ErrorContext(c, "Got null getting existResponse", logger.AttributeError, err)
			return nil
		}

		existsMap := utils.ToMap(existResponse)

		for _, cp := range chatParticipants {
			ue, ok := existsMap[cp.UserId]
			if !ok {
				srv.lgr.WarnContext(c, "aaa responded no exists, probably the error in aaa", logger.AttributeUserId, cp.UserId)
				continue
			}

			if !ue.Exists {
				srv.lgr.InfoContext(c, "Deleting participant because it does not exists in aaa", logger.AttributeUserId, ue.UserId, logger.AttributeChatId, cp.ChatId)
				cmd := cqrs.TechnicalRemoveContentOfDeletedUser{ // ~ DeleteParticipant
					UserId: cp.UserId,
					ChatId: cp.ChatId,
				}

				err = cmd.Handle(c, srv.eventBus)
				if err != nil {
					srv.lgr.ErrorContext(c, "error during removing content of deleted user", logger.AttributeError, err)
				}
			}
		}

		return nil
	})
	if errOuter != nil {
		srv.lgr.ErrorContext(c, "error during removing content of deleted user", logger.AttributeError, errOuter)
	}

	srv.lgr.InfoContext(c, "End of cleaning deleted users data job")
}

func (srv *CleanDeletedUserDataService) spanStarter(ctx context.Context) (context.Context, any) {
	return srv.tracer.Start(ctx, "scheduler.cleanDeletedUsersData")
}

func (srv *CleanDeletedUserDataService) spanFinisher(ctx context.Context, span any) {
	span.(trace.Span).End()
}

func NewCleanDeletedUserDataService(
	lgr *logger.LoggerWrapper,
	chatClient client.AaaRestClient,
	dbR *db.DB,
	eventBus *cqrs.KafkaProducer,
	co *cqrs.CommonProjection,
) *CleanDeletedUserDataService {
	trcr := otel.Tracer("scheduler/clean-deleted-users-data")
	return &CleanDeletedUserDataService{
		restClient: chatClient,
		tracer:     trcr,
		dbR:        dbR,
		lgr:        lgr,
		eventBus:   eventBus,
		co:         co,
	}
}
