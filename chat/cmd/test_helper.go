package cmd

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

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

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
)

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	shutdown()
	os.Exit(retCode)
}

func setup() {

}

func shutdown() {

}

func resetInfra(lgr *logger.LoggerWrapper, cfg *config.AppConfig) {
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
			db.ConfigureDatabase,
			kafka.ConfigureKafkaAdmin,
			rabbitmq.CreateRabbitMqConnection,
		),
		fx.Invoke(
			db.RunResetDatabaseHard,
			kafka.RunDeleteTopicChat,
			kafka.RunDeleteTopicUser,
			listener.DeleteTestEventQueue,
			db.RunMigrations,
			kafka.RunCreateTopicChat,
			kafka.RunCreateTopicUser,
			app.Shutdown,
		),
	)
	appFx.Run()
}

func aaaClientFactory(t *testing.T) func() client.AaaRestClient {
	return func() client.AaaRestClient {
		return client.NewMockAaaRestClient(t)
	}
}

func runTestFunc(lgr *logger.LoggerWrapper, cfg *config.AppConfig, t *testing.T, testFunc interface{}) {
	var s fx.Shutdowner

	appTestFx := fxtest.New(
		t,
		fx.Supply(cfg),
		fx.Supply(lgr),
		fx.WithLogger(func(lgr *logger.LoggerWrapper) fxevent.Logger {
			fsl := &fxevent.SlogLogger{Logger: lgr.Logger}
			fsl.UseLogLevel(slog.LevelDebug)
			return fsl
		}),
		fx.Populate(&s),
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
			client.NewTestRestClient,
			aaaClientFactory(t),
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
			producer.NewRabbitTestInputEventsPublisher,
			rabbitmq.CreateRabbitMqConnection,
			cqrs.NewEventHandler,
			listener.CreateRabbitTestOutputEventListener,
			listener.CreateRabbitTestNotificationEventListener,
			listener.NewRabbitTestOutputEventAccumulator,
			listener.NewRabbitTestNotificationEventAccumulator,
			listener.CreateRabbitInternalEventsListener,
			listener.CreateRabbitAaaUserProfileUpdateListener,
			type_registry.NewTypeRegistryInstance,
			tasks.NewCleanAbandonedChatsService,
			tasks.NewCleanDeletedUserDataService,
			cqrs.NewKafkaListener,
			cqrs.NewBatchOptimizer,
		),
		fx.Invoke(
			cqrs.ListenChatTopic,
			cqrs.ListenUserTopic,
			producer.EnableOutputEvents,
			producer.EnableNotificationEvents,
			listener.CreateAndListenTestOutputEventChannel,
			listener.CreateAndListenTestNotificationEventChannel,
			listener.CreateAndListenInternalEventsChannel,
			listener.CreateAndListenAaaChannel,
			cqrs.UnsetIsNeedToSkipImport,
			handlers.RunHttpServer,
			waitForHealthCheck,
			testFunc,
		),
	)
	defer appTestFx.RequireStart().RequireStop()
	assert.NoError(t, s.Shutdown(), "error in app shutdown")
}

func startAppFull(t *testing.T, testFunc interface{}) {
	cfg, err := config.CreateTestTypedConfig()
	if err != nil {
		panic(err)
	}
	lgr := logger.NewLogger(os.Stdout, cfg)
	defer lgr.CloseLogger()

	resetInfra(lgr, cfg)

	runTestFunc(lgr, cfg, t, testFunc)
}

func waitForHealthCheck(lgr *logger.LoggerWrapper, restClient *client.TestRestClient, cfg *config.AppConfig) {
	ctx := context.Background()

	i := 0
	success := false
	for ; i <= cfg.Cqrs.PollingMaxTimes; i++ {
		err := restClient.HealthCheck(ctx)
		if err != nil {
			lgr.Info("Awaiting while chat have been started")
			time.Sleep(cfg.Cqrs.SleepBeforePolling)
			continue
		} else {
			success = true
			break
		}
	}
	if !success {
		panic("Cannot await for chat will be started")
	}
	lgr.Info("chat have started")
}

func waitForMessageExists(lgr *logger.LoggerWrapper, commonProjection *cqrs.CommonProjection, dba *db.DB, chatId, messageId int64, sleepBeforePolling time.Duration, maxAttempts int) {
	ctx := context.Background()

	i := 0
	success := false
	for ; i <= maxAttempts; i++ {
		exists, err := commonProjection.IsMessageExists(ctx, dba, chatId, messageId)
		if err != nil || !exists {
			lgr.Info("Awaiting while message appear")
			time.Sleep(sleepBeforePolling)
			continue
		} else {
			success = true
			break
		}
	}
	if !success {
		panic("Cannot await for message will appear")
	}
	lgr.Info("message appeared")
}

func waitForChatExists(lgr *logger.LoggerWrapper, commonProjection *cqrs.CommonProjection, dba *db.DB, chatId, behalfUserId int64, sleepBeforePolling time.Duration, maxAttempts int) {
	ctx := context.Background()

	i := 0
	success := false
	for ; i <= maxAttempts; i++ {

		exists, err := commonProjection.IsChatUserViewExists(ctx, dba, chatId, behalfUserId)
		if err != nil || !exists {
			lgr.Info("Awaiting while chat appear")
			time.Sleep(sleepBeforePolling)
			continue
		} else {
			success = true
			break
		}
	}
	if !success {
		panic("Cannot await for chat will appear")
	}
	lgr.Info("chat appeared")
}

func waitForChatNotExists(lgr *logger.LoggerWrapper, commonProjection *cqrs.CommonProjection, dba *db.DB, chatId int64, sleepBeforePolling time.Duration, maxAttempts int) {
	ctx := context.Background()

	i := 0
	success := false
	for ; i <= maxAttempts; i++ {

		exists, err := commonProjection.IsChatExists(ctx, dba, chatId)
		if err != nil || exists {
			lgr.Info("Awaiting while chat disappear")
			time.Sleep(sleepBeforePolling)
			continue
		} else {
			success = true
			break
		}
	}
	if !success {
		panic("Cannot await for chat will disappear")
	}
	lgr.Info("chat disappeared")
}
