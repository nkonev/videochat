package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nkonev/dcron"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	jaegerPropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"nkonev.name/video/app"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	"nkonev.name/video/db"
	"nkonev.name/video/handlers"
	"nkonev.name/video/listener"
	"nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/rabbitmq"
	"nkonev.name/video/services"
	"nkonev.name/video/tasks"
	"nkonev.name/video/type_registry"
)

const EXTERNAL_TRACE_ID_HEADER = "trace-id"
const TRACE_RESOURCE = app.APP_NAME

func main() {
	config.InitViper()
	lgr := logger.NewLogger()
	defer lgr.CloseLogger()

	appFx := fx.New(
		fx.Supply(lgr),
		fx.WithLogger(func(log *logger.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log.ZapLogger}
		}),
		fx.Provide(
			createTypedConfig,
			configureTracer,
			configureApiEcho,
			client.NewRestClient,
			client.NewLivekitClient,
			client.NewEgressClient,
			handlers.NewUserHandler,
			handlers.NewConfigHandler,
			handlers.ConfigureApiStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			handlers.NewLivekitWebhookHandler,
			handlers.NewInviteHandler,
			handlers.NewRecordHandler,
			rabbitmq.CreateRabbitMqConnection,
			producer.NewRabbitUserCountPublisher,
			producer.NewRabbitInvitePublisher,
			producer.NewRabbitUserIdsPublisher,
			producer.NewRabbitDialStatusPublisher,
			producer.NewRabbitRecordingPublisher,
			producer.NewRabbitNotificationsPublisher,
			producer.NewRabbitScreenSharePublisher,
			services.NewNotificationService,
			services.NewUserService,
			services.NewStateChangedEventService,
			services.NewEgressService,
			tasks.RedisV9,
			tasks.RedisLocker,
			tasks.Scheduler,
			tasks.NewVideoCallUsersCountNotifierService,
			tasks.VideoCallUsersCountNotifierScheduler,
			tasks.NewUsersInVideoStatusNotifierService,
			tasks.UsersInVideoStatusNotifierScheduler,
			tasks.NewChatDialerService,
			tasks.ChatDialerScheduler,
			tasks.NewRecordingNotifierService,
			tasks.RecordingNotifierScheduler,
			tasks.NewSynchronizeWithLivekitService,
			tasks.SynchronizeWithLivekitSheduler,
			listener.CreateAaaUserSessionsKilledListener,
			type_registry.NewTypeRegistryInstance,
			configureMigrations,
			db.ConfigureDb,
		),
		fx.Invoke(
			runMigrations,
			runApiEcho,
			runScheduler,
			listener.CreateAaaChannel,
		),
	)
	appFx.Run()

	lgr.Infof("Exit program")
}

func configureWriteHeaderMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			existsSpan := trace.SpanFromContext(c.Request().Context())
			if existsSpan.SpanContext().HasTraceID() {
				c.Response().Header().Set(EXTERNAL_TRACE_ID_HEADER, existsSpan.SpanContext().TraceID().String())
			}
			if err := next(c); err != nil {
				c.Error(err)
			}
			return nil
		}
	}
}

func configureOpentelemetryMiddleware(tp *sdktrace.TracerProvider) echo.MiddlewareFunc {
	mw := otelecho.Middleware(TRACE_RESOURCE, otelecho.WithTracerProvider(tp))
	return mw
}

func createCustomHTTPErrorHandler(lgr *logger.Logger, e *echo.Echo) func(err error, c echo.Context) {
	originalHandler := e.DefaultHTTPErrorHandler
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		formattedStr := eris.ToString(err, true)
		lgr.WithTracing(c.Request().Context()).Errorf("Unhandled error: %v", formattedStr)
		originalHandler(err, c)
	}
}

type ApiEcho struct {
	*echo.Echo
}

func configureApiEcho(
	lgr *logger.Logger,
	cfg *config.ExtendedConfig,
	authMiddleware handlers.AuthMiddleware,
	staticMiddleware handlers.ApiStaticMiddleware,
	lc fx.Lifecycle,
	uh *handlers.UserHandler,
	ch *handlers.ConfigHandler,
	lhf *handlers.LivekitWebhookHandler,
	ih *handlers.InviteHandler,
	rh *handlers.RecordHandler,
	tp *sdktrace.TracerProvider,
) *ApiEcho {

	bodyLimit := cfg.HttpServerConfig.BodyLimit

	e := echo.New()
	e.Logger.SetOutput(lgr)

	e.HTTPErrorHandler = createCustomHTTPErrorHandler(lgr, e)

	e.Pre(echo.MiddlewareFunc(staticMiddleware))
	e.Use(configureOpentelemetryMiddleware(tp))
	skipper := func(c echo.Context) bool {
		// Skip health check endpoint
		return c.Request().URL.Path == "/health"
	}
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:        true,
		LogURI:           true,
		LogMethod:        true,
		LogRemoteIP:      true,
		LogError:         true,
		LogLatency:       true,
		LogUserAgent:     true,
		LogContentLength: true,
		LogResponseSize:  true,
		Skipper:          skipper,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			tl := lgr.SugaredLogger
			spanCtx := trace.SpanContextFromContext(c.Request().Context())
			if spanCtx.HasTraceID() {
				tl = lgr.With(
					zap.String("trace_id", spanCtx.TraceID().String()),
					zap.String("span_id", spanCtx.SpanID().String()),
				)
			}
			tl = tl.With(
				"status", v.Status,
				"uri", v.URI,
				"method", v.Method,
				"remote_ip", v.RemoteIP,
				"latency", v.Latency,
				"user_agent", v.UserAgent,
				"content_length", v.ContentLength,
				"response_size", v.ResponseSize,
			)

			if v.Error == nil {
				tl.Infof("REQUEST")
			} else {
				tl = tl.With(
					"error", v.Error.Error(),
				)
				tl.Errorf("REQUEST")
			}
			return nil
		},
	}))
	e.Use(configureWriteHeaderMiddleware())
	e.Use(echo.MiddlewareFunc(authMiddleware))
	e.Use(middleware.Secure())
	e.Use(middleware.BodyLimit(bodyLimit))

	e.GET("/api/video/:chatId/users", uh.GetVideoUsers)
	e.GET("/api/video/config", ch.GetConfig)
	e.POST("/internal/livekit-webhook", lhf.GetLivekitWebhookHandler())
	e.PUT("/api/video/:chatId/kick", uh.Kick)
	e.PUT("/api/video/:chatId/mute", uh.Mute)

	e.PUT("/api/video/:id/dial/invite", ih.ProcessCreatingOrDeletingInvite) // used by owner to add or remove from dial list
	e.PUT("/api/video/:id/dial/enter", ih.ProcessEnterToDial)               // user enters to call somehow, either by clicking green tube or opening .../video link
	e.PUT("/api/video/:id/dial/cancel", ih.ProcessCancelInvitation)         // cancelling by invitee
	e.PUT("/api/video/:id/dial/exit", ih.ProcessExit)                       // used by any user on exit
	e.PUT("/api/video/user/request-in-video-status", ih.SendCurrentInVideoStatuses)
	e.GET("/api/video/user/being-invited-status", ih.GetMyBeingInvitedStatus)

	e.PUT("/api/video/:id/record/start", rh.StartRecording)
	e.PUT("/api/video/:id/record/stop", rh.StopRecording)
	e.GET("/api/video/:id/record/status", rh.StatusRecording)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// do some work on application stop (like closing connections and files)
			lgr.Infof("Stopping http server")
			return e.Shutdown(ctx)
		},
	})

	return &ApiEcho{e}
}

func configureTracer(lgr *logger.Logger, lc fx.Lifecycle, cfg *config.ExtendedConfig) (*sdktrace.TracerProvider, error) {
	lgr.Infof("Configuring Jaeger tracing")
	conn, err := grpc.DialContext(context.Background(), cfg.OtlpConfig.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	exporter, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}
	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(TRACE_RESOURCE),
	)
	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(batchSpanProcessor),
		sdktrace.WithResource(resources),
	)
	otel.SetTracerProvider(tp)
	aJaegerPropagator := jaegerPropagator.Jaeger{}
	// register jaeger propagator
	otel.SetTextMapPropagator(aJaegerPropagator)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Infof("Stopping tracer")
			if err := tp.Shutdown(context.Background()); err != nil {
				lgr.Errorf("Error shutting down tracer provider: %v", err)
			}
			return nil
		},
	})

	return tp, nil
}

func configureMigrations() *db.MigrationsConfig {
	return &db.MigrationsConfig{}
}

func runMigrations(lgr *logger.Logger, db *db.DB, migrationsConfig *db.MigrationsConfig) {
	db.Migrate(lgr, migrationsConfig)
}

// rely on viper import and it's configured by
func runApiEcho(lgr *logger.Logger, e *ApiEcho, cfg *config.ExtendedConfig) {
	address := cfg.HttpServerConfig.ApiAddress

	lgr.Info("Starting api server...")
	// Start server in another goroutine
	go func() {
		if err := e.Start(address); err != nil {
			lgr.Infof("server shut down: %v", err)
		}
	}()
	lgr.Info("Api server started. Waiting for interrupt signal 2 (Ctrl+C)")
}

func runScheduler(
	lgr *logger.Logger,
	scheduler *dcron.Cron,
	chatNotifierTask *tasks.VideoCallUsersCountNotifierTask,
	chatDialerTask *tasks.ChatDialerTask,
	videoRecordingTask *tasks.RecordingNotifierTask,
	usersInVideoStatusNotifierTask *tasks.UsersInVideoStatusNotifierTask,
	synchronizeWithLivekitTask *tasks.SynchronizeWithLivekitTask,
	lc fx.Lifecycle,
) error {
	scheduler.Start()
	lgr.Infof("Scheduler started")

	if viper.GetBool("schedulers." + chatNotifierTask.Key() + ".enabled") {
		lgr.Infof("Adding task " + chatNotifierTask.Key() + " to scheduler")
		err := scheduler.AddJobs(chatNotifierTask)
		if err != nil {
			return err
		}
	} else {
		lgr.Infof("Task " + chatNotifierTask.Key() + " is disabled")
	}

	if viper.GetBool("schedulers." + chatDialerTask.Key() + ".enabled") {
		lgr.Infof("Adding task " + chatDialerTask.Key() + " to scheduler")
		err := scheduler.AddJobs(chatDialerTask)
		if err != nil {
			return err
		}
	} else {
		lgr.Infof("Task " + chatDialerTask.Key() + " is disabled")
	}

	if viper.GetBool("schedulers." + videoRecordingTask.Key() + ".enabled") {
		lgr.Infof("Adding task " + videoRecordingTask.Key() + " to scheduler")
		err := scheduler.AddJobs(videoRecordingTask)
		if err != nil {
			return err
		}
	} else {
		lgr.Infof("Task " + videoRecordingTask.Key() + " is disabled")
	}

	if viper.GetBool("schedulers." + usersInVideoStatusNotifierTask.Key() + ".enabled") {
		lgr.Infof("Adding task " + usersInVideoStatusNotifierTask.Key() + " to scheduler")
		err := scheduler.AddJobs(usersInVideoStatusNotifierTask)
		if err != nil {
			return err
		}
	} else {
		lgr.Infof("Task " + usersInVideoStatusNotifierTask.Key() + " is disabled")
	}

	if viper.GetBool("schedulers." + synchronizeWithLivekitTask.Key() + ".enabled") {
		lgr.Infof("Adding task " + synchronizeWithLivekitTask.Key() + " to scheduler")
		err := scheduler.AddJobs(synchronizeWithLivekitTask)
		if err != nil {
			return err
		}
	} else {
		lgr.Infof("Task " + synchronizeWithLivekitTask.Key() + " is disabled")
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Infof("Stopping scheduler")
			<-scheduler.Stop().Done()
			return nil
		},
	})
	return nil
}

func createTypedConfig() (*config.ExtendedConfig, error) {
	conf := config.ExtendedConfig{}
	err := viper.GetViper().Unmarshal(&conf)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("sfu extended config file loaded failed. %v\n", err))
	}

	return &conf, nil
}
