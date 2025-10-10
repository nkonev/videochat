package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"nkonev.name/chat/app"
	"nkonev.name/chat/client"
	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers"
	"nkonev.name/chat/kafka"
	"nkonev.name/chat/listener"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/otel"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/sanitizer"
	"nkonev.name/chat/services"
	"nkonev.name/chat/tasks"
	"nkonev.name/chat/type_registry"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

const CommandServeName = "serve"

func RunServe(args []string) {
	processedArgs, hasHelp := app.IsHelp(args)
	if hasHelp {
		fmt.Printf(`
Starts normal api requests serving.
Http server starts when all the events from the Kafka events topic were consumed and
the 'need_to_fast_forward_sequences' task
in "technical" PostgreSQL table is finished.
Also starts schedulers and RabbitMQ listeners.

To run with config:
./%s %s %s=/path/to/config.yaml

To run with override log level:
./%s %s --logger.level=debug

Or via environment variable:
CHAT_LOGGER_LEVEL=debug ./%s %s

To run with override log json:
./%s %s --logger.json=false

To run on the specific port:
./%s %s --server.address=:8888

To run without schedulers:
./%s %s --schedulers.cleanAbandonedChatsTask.enabled=false --schedulers.cleanDeletedUsersDataTask.enabled=false

`, ExecutableName, CommandServeName, app.ConfigLongPrefix,
			ExecutableName, CommandServeName,
			ExecutableName, CommandServeName,
			ExecutableName, CommandServeName,
			ExecutableName, CommandServeName,
			ExecutableName, CommandServeName,
		)

		return
	}

	cfg, err := config.CreateTypedConfig(processedArgs)
	if err != nil {
		panic(err)
	}
	lgr := logger.NewLogger(os.Stdout, cfg)
	defer lgr.CloseLogger()

	lgr.Info("Start serve command")

	appFx := fx.New(
		fx.Supply(cfg),
		fx.Supply(lgr),
		fx.WithLogger(func(lgr *logger.LoggerWrapper) fxevent.Logger {
			fsl := &fxevent.SlogLogger{Logger: lgr.Logger}
			fsl.UseLogLevel(slog.LevelDebug)
			return fsl
		}),
		fx.Provide(
			otel.ConfigureTracePropagator,
			otel.ConfigureTraceProvider,
			otel.ConfigureTraceExporter,
			cqrs.NewKotelTracer,
			cqrs.NewKotel,
			db.ConfigureDatabase,
			kafka.ConfigureKafkaAdmin,
			cqrs.ConfigurePublisher,
			cqrs.ConfigureCommonProjection,
			cqrs.NewEnrichingProjection,
			handlers.NewChatHandler,
			handlers.NewParticipantHandler,
			handlers.NewMessageHandler,
			handlers.NewBlogHandler,
			handlers.NewTechnicalHandler,
			handlers.NewStaticHandler,
			handlers.CreateHttpRouter,
			handlers.ConfigureHttpServer,
			client.NewAAARestClient,
			sanitizer.CreateSanitizer,
			sanitizer.CreateStripTags,
			sanitizer.CreateStripSource,
			services.NewAuthorizationService,
			services.NewMessageService,
			services.NewAsyncMessageService,
			services.NewInputEventHandler,
			producer.NewRabbitOutputEventsPublisher,
			producer.NewRabbitNotificationEventsPublisher,
			producer.NewRabbitInternalEventsPublisher,
			rabbitmq.CreateRabbitMqConnection,
			cqrs.NewEventHandler,
			listener.CreateRabbitInternalEventsListener,
			listener.CreateRabbitAaaUserProfileUpdateListener,
			type_registry.NewTypeRegistryInstance,
			tasks.RedisV9,
			tasks.RedisLocker,
			tasks.Scheduler,
			tasks.CleanAbandonedChatsScheduler,
			tasks.CleanDeletedUserDataScheduler,
			tasks.NewCleanAbandonedChatsService,
			tasks.NewCleanDeletedUserDataService,
			cqrs.NewKafkaListener,
			cqrs.NewBatchOptimizer,
		),
		fx.Invoke(
			db.RunMigrations,
			kafka.RunCreateTopicChat,
			kafka.RunCreateTopicUser,
			cqrs.ListenChatTopic,
			cqrs.ListenUserTopic,
			kafka.WaitForAllEventsProcessedChat,
			kafka.WaitForAllEventsProcessedUser,
			cqrs.RunSequenceFastforwarder,
			producer.EnableOutputEvents,
			producer.EnableNotificationEvents,
			listener.CreateAndListenInternalEventsChannel,
			listener.CreateAndListenAaaChannel,
			tasks.RunScheduler,
			handlers.RunHttpServer,
		),
	)
	appFx.Run()
	lgr.Info("Exit serve command")
}
