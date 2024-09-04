package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	"nkonev.name/video/db"
	"nkonev.name/video/handlers"
	"nkonev.name/video/listener"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/rabbitmq"
	"nkonev.name/video/services"
	"nkonev.name/video/tasks"
	"nkonev.name/video/type_registry"
)

const EXTERNAL_TRACE_ID_HEADER = "trace-id"
const TRACE_RESOURCE = "video"

func main() {
	config.InitViper()

	app := fx.New(
		fx.Logger(Logger),
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
			tasks.RedisV8,
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
	app.Run()

	Logger.Infof("Exit program")
}

func configureWriteHeaderMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) (err error) {
			handler := http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ctx.SetRequest(r)
					ctx.SetResponse(echo.NewResponse(w, ctx.Echo()))
					existsSpan := trace.SpanFromContext(ctx.Request().Context())
					if existsSpan.SpanContext().HasTraceID() {
						w.Header().Set(EXTERNAL_TRACE_ID_HEADER, existsSpan.SpanContext().TraceID().String())
					}
					err = next(ctx)
				},
			)
			handler.ServeHTTP(ctx.Response(), ctx.Request())
			return
		}
	}
}

func configureOpentelemetryMiddleware(tp *sdktrace.TracerProvider) echo.MiddlewareFunc {
	mw := otelecho.Middleware(TRACE_RESOURCE, otelecho.WithTracerProvider(tp))
	return mw
}

func createCustomHTTPErrorHandler(e *echo.Echo) func(err error, c echo.Context) {
	originalHandler := e.DefaultHTTPErrorHandler
	return func(err error, c echo.Context) {
		formattedStr := eris.ToString(err, true)
		GetLogEntry(c.Request().Context()).Errorf("Unhandled error: %v", formattedStr)
		originalHandler(err, c)
	}
}

type ApiEcho struct {
	*echo.Echo
}

func configureApiEcho(
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
	e.Logger.SetOutput(Logger.Writer())

	e.HTTPErrorHandler = createCustomHTTPErrorHandler(e)

	e.Pre(echo.MiddlewareFunc(staticMiddleware))
	e.Use(configureOpentelemetryMiddleware(tp))
	e.Use(configureWriteHeaderMiddleware())
	e.Use(echo.MiddlewareFunc(authMiddleware))
	accessLoggerConfig := middleware.LoggerConfig{
		Output: Logger.Writer(),
		Format: `"remote_ip":"${remote_ip}",` +
			`"method":"${method}","uri":"${uri}",` +
			`"status":${status},` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out},"traceId":"${header:uber-trace-id}"` + "\n",
	}
	e.Use(middleware.LoggerWithConfig(accessLoggerConfig))
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
			Logger.Infof("Stopping http server")
			return e.Shutdown(ctx)
		},
	})

	return &ApiEcho{e}
}

func configureTracer(lc fx.Lifecycle, cfg *config.ExtendedConfig) (*sdktrace.TracerProvider, error) {
	Logger.Infof("Configuring Jaeger tracing")
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
			Logger.Infof("Stopping tracer")
			if err := tp.Shutdown(context.Background()); err != nil {
				Logger.Printf("Error shutting down tracer provider: %v", err)
			}
			return nil
		},
	})

	return tp, nil
}

func configureMigrations() *db.MigrationsConfig {
	return &db.MigrationsConfig{}
}

func runMigrations(db *db.DB, migrationsConfig *db.MigrationsConfig) {
	db.Migrate(migrationsConfig)
}

// rely on viper import and it's configured by
func runApiEcho(e *ApiEcho, cfg *config.ExtendedConfig) {
	address := cfg.HttpServerConfig.ApiAddress

	Logger.Info("Starting api server...")
	// Start server in another goroutine
	go func() {
		if err := e.Start(address); err != nil {
			Logger.Infof("server shut down: %v", err)
		}
	}()
	Logger.Info("Api server started. Waiting for interrupt signal 2 (Ctrl+C)")
}

func runScheduler(
	chatNotifierTask *tasks.VideoCallUsersCountNotifierTask,
	chatDialerTask *tasks.ChatDialerTask,
	videoRecordingTask *tasks.RecordingNotifierTask,
	usersInVideoStatusNotifierTask *tasks.UsersInVideoStatusNotifierTask,
	synchronizeWithLivekitTask *tasks.SynchronizeWithLivekitTask,
) {
	if viper.GetBool("schedulers.videoCallUsersCountNotifierTask.enabled") {
		go func() {
			Logger.Infof("Starting scheduler videoCallUsersCountNotifierTask")
			err := chatNotifierTask.Run(context.Background())
			if err != nil {
				Logger.Errorf("Error during working videoCallUsersCountNotifierTask: %s", err)
			}
		}()
	}
	if viper.GetBool("schedulers.chatDialerTask.enabled") {
		go func() {
			Logger.Infof("Starting scheduler chatDialerTask")
			err := chatDialerTask.Run(context.Background())
			if err != nil {
				Logger.Errorf("Error during working chatDialerTask: %s", err)
			}
		}()
	}
	if viper.GetBool("schedulers.videoRecordingNotifierTask.enabled") {
		go func() {
			Logger.Infof("Starting scheduler videoRecordingNotifierTask")
			err := videoRecordingTask.Run(context.Background())
			if err != nil {
				Logger.Errorf("Error during working videoRecordingNotifierTask: %s", err)
			}
		}()
	}
	if viper.GetBool("schedulers.usersInVideoStatusNotifierTask.enabled") {
		go func() {
			Logger.Infof("Starting scheduler usersInVideoStatusNotifierTask")
			err := usersInVideoStatusNotifierTask.Run(context.Background())
			if err != nil {
				Logger.Errorf("Error during working usersInVideoStatusNotifierTask: %s", err)
			}
		}()
	}
	if viper.GetBool("schedulers.synchronizeWithLivekitTask.enabled") {
		go func() {
			Logger.Infof("Starting scheduler synchronizeWithLivekitTask")
			err := synchronizeWithLivekitTask.Run(context.Background())
			if err != nil {
				Logger.Errorf("Error during working synchronizeWithLivekitTask: %s", err)
			}
		}()
	}
}

func createTypedConfig() (*config.ExtendedConfig, error) {
	conf := config.ExtendedConfig{}
	err := viper.GetViper().Unmarshal(&conf)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("sfu extended config file loaded failed. %v\n", err))
	}

	return &conf, nil
}
