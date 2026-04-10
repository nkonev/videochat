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
	"nkonev.name/chat/kafka"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/otel"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/sanitizer"
	"nkonev.name/chat/type_registry"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

const CommandRewindName = "rewind"

func RunRewind(args []string) {
	processedArgs, hasHelp := app.IsHelp(args)
	if hasHelp {
		fmt.Printf(`
Consumes all the events from the Kafka events topic
hereby (re)building PostgreSQL projections
and processes the 'need_to_fast_forward_sequences' task
in "technical" PostgreSQL table.
Then exits.

./%s %s --rabbitmq.skipPublishOutputEventsOnRewind=true --rabbitmq.skipPublishNotificationEventsOnRewind=true
`, ExecutableName, CommandRewindName)

		return
	}

	cfg, err := config.CreateTypedConfig(processedArgs)
	if err != nil {
		panic(err)
	}
	lgr := logger.NewLogger(os.Stdout, cfg)
	defer lgr.CloseLogger()

	lgr.Info("Start rewind command")

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
			client.NewAAARestClient,
			sanitizer.CreateSanitizer,
			sanitizer.CreateStripTags,
			sanitizer.CreateStripSource,
			producer.NewRabbitOutputEventsPublisher,
			producer.NewRabbitNotificationEventsPublisher,
			rabbitmq.CreateRabbitMqConnection,
			cqrs.NewEventHandler,
			type_registry.NewTypeRegistryInstance,
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
			cqrs.UnsetIsNeedToSkipImport,
			cqrs.RunSequenceFastforwarder,
			app.Shutdown,
		),
	)
	appFx.Run()
	lgr.Info("Exit rewind command")
}
