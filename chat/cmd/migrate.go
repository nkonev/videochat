package cmd

import (
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
	"nkonev.name/chat/services"
	"nkonev.name/chat/type_registry"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

const CommandMigrateName = "migrate"

func RunMigrate(args []string) {
	cfg, err := config.CreateTypedConfig(args)
	if err != nil {
		panic(err)
	}
	lgr := logger.NewLogger(os.Stdout, cfg)
	defer lgr.CloseLogger()

	lgr.Info("Start migrate command")

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
			services.NewAuthorizationService,
			services.NewMessageService,
			services.NewAsyncMessageService,
			services.NewInputEventHandler,
			producer.NewRabbitOutputEventsPublisher,
			producer.NewRabbitNotificationEventsPublisher,
			producer.NewRabbitInternalEventsPublisher,
			rabbitmq.CreateRabbitMqConnection,
			type_registry.NewTypeRegistryInstance,
		),
		fx.Invoke(
			db.RunMigrations,
			kafka.RunCreateTopicChat,
			kafka.RunCreateTopicUser,
			cqrs.RunMigrateFromOldDb,
			app.Shutdown,
		),
	)
	appFx.Run()
	lgr.Info("Exit migrate command")
}
