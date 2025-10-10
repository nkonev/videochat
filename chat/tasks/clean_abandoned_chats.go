package tasks

import (
	"context"
	"nkonev.name/chat/client"
	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/logger"

	"github.com/nkonev/dcron"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const CleanAbandonedChatsSchedulerKey = "cleanAbandonedChatsTask"

type CleanAbandonedChatsTask struct {
	dcron.Job
}

func CleanAbandonedChatsScheduler(
	lgr *logger.LoggerWrapper,
	service *CleanAnandonedChatsService,
	cfg *config.AppConfig,
) *CleanAbandonedChatsTask {
	var str = cfg.Schedulers.CleanAbandonedChatsTask.Cron
	lgr.Info("Created CleanAbandonedChatsScheduler with cron", "cron", str, dcron.SlogKeyTaskName, CleanAbandonedChatsSchedulerKey)

	job := dcron.NewJob(CleanAbandonedChatsSchedulerKey, str, func(ctx context.Context) error {
		service.DoJob(ctx)
		return nil
	},
		dcron.WithTracing(service.spanStarter, service.spanFinisher),
		dcron.WithJobSettings(cfg.Schedulers.CleanAbandonedChatsTask.Expiration),
	)

	return &CleanAbandonedChatsTask{job}
}

type CleanAnandonedChatsService struct {
	restClient client.AaaRestClient
	tracer     trace.Tracer
	dbR        *db.DB
	lgr        *logger.LoggerWrapper
	eventBus   *cqrs.KafkaProducer
	co         *cqrs.CommonProjection
}

func (srv *CleanAnandonedChatsService) DoJob(ctx context.Context) {
	srv.processChats(ctx)
}

func (srv *CleanAnandonedChatsService) processChats(c context.Context) {
	srv.lgr.InfoContext(c, "Starting cleaning abandoned chats job")

	errOuter := srv.co.IterateOverAllChats(c, srv.dbR, func(chatIdsPortion []int64) error {
		hasParticipantsMap, err := srv.co.HasParticipants(c, srv.dbR, chatIdsPortion) // will re-check on the projection side after kafka
		if err != nil {
			srv.lgr.ErrorContext(c, "Got error HasParticipants", logger.AttributeError, err)
			return nil
		}

		for _, ch := range chatIdsPortion {
			hasParticipants := hasParticipantsMap[ch]

			if !hasParticipants {
				srv.lgr.InfoContext(c, "Deleting chat because it does not have participants", logger.AttributeChatId, ch)
				cmd := cqrs.TechnicalRemoveAbandonedChat{
					ChatId: ch,
				}
				err = cmd.Handle(c, srv.eventBus)
				if err != nil {
					srv.lgr.ErrorContext(c, "error during removing abandoned chats", logger.AttributeError, err)
				}
			}
		}
		return nil
	})
	if errOuter != nil {
		srv.lgr.ErrorContext(c, "error during removing abandoned chats", logger.AttributeError, errOuter)
	}

	srv.lgr.InfoContext(c, "End of cleaning abandoned chats job")
}

func (srv *CleanAnandonedChatsService) spanStarter(ctx context.Context) (context.Context, any) {
	return srv.tracer.Start(ctx, "scheduler.cleanAbandonedChats")
}

func (srv *CleanAnandonedChatsService) spanFinisher(ctx context.Context, span any) {
	span.(trace.Span).End()
}

func NewCleanAbandonedChatsService(
	lgr *logger.LoggerWrapper,
	chatClient client.AaaRestClient,
	dbR *db.DB,
	eventBus *cqrs.KafkaProducer,
	co *cqrs.CommonProjection,
) *CleanAnandonedChatsService {
	trcr := otel.Tracer("scheduler/clean-abandoned-chats")
	return &CleanAnandonedChatsService{
		restClient: chatClient,
		tracer:     trcr,
		dbR:        dbR,
		lgr:        lgr,
		eventBus:   eventBus,
		co:         co,
	}
}
