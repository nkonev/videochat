package main

import (
	"context"
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
	"nkonev.name/chat/app"
	"nkonev.name/chat/client"
	"nkonev.name/chat/config"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers"
	"nkonev.name/chat/listener"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/services"
	"nkonev.name/chat/tasks"
	"nkonev.name/chat/type_registry"
)

const EXTERNAL_TRACE_ID_HEADER = "trace-id"
const TRACE_RESOURCE = app.APP_NAME

func main() {
	config.InitViper()
	lgr := logger.NewLogger()

	appFx := fx.New(
		fx.Supply(lgr),
		fx.WithLogger(func(log *logger.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log.ZapLogger}
		}),
		fx.Provide(
			configureTracer,
			client.NewRestClient,
			services.CreateSanitizer,
			services.CreateStripTags,
			services.StripStripSourcePolicy,
			handlers.NewChatHandler,
			handlers.NewMessageHandler,
			handlers.NewBlogHandler,
			configureEcho,
			handlers.ConfigureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			configureMigrations,
			db.ConfigureDb,
			tasks.RedisV9,
			tasks.RedisLocker,
			tasks.Scheduler,
			tasks.CleanChatsOfDeletedUserScheduler,
			tasks.NewCleanChatsOfDeletedUserService,
			services.NewEvents,
			producer.NewRabbitEventsPublisher,
			producer.NewRabbitNotificationsPublisher,
			listener.CreateAaaUserProfileUpdateListener,
			rabbitmq.CreateRabbitMqConnection,
			type_registry.NewTypeRegistryInstance,
		),
		fx.Invoke(
			runMigrations,
			runScheduler,
			runEcho,
			listener.CreateAaaChannel,
		),
	)
	appFx.Run()

	lgr.Infof("Exit program")
	lgr.CloseLogger()
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

func configureEcho(
	lgr *logger.Logger,
	staticMiddleware handlers.StaticMiddleware,
	authMiddleware handlers.AuthMiddleware,
	lc fx.Lifecycle,
	ch *handlers.ChatHandler,
	mc *handlers.MessageHandler,
	bh *handlers.BlogHandler,
	tp *sdktrace.TracerProvider,
) *echo.Echo {

	bodyLimit := viper.GetString("server.body.limit")

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

	e.POST("/api/chat/search", ch.GetChats)
	e.GET("/api/chat/has-new-messages", ch.HasNewMessages)
	e.POST("/api/chat/filter", ch.Filter)
	e.GET("/api/chat/:id", ch.GetChat)
	e.POST("/api/chat/fresh", ch.IsFreshChatsPage)
	e.POST("/api/chat", ch.CreateChat)
	e.DELETE("/api/chat/:id", ch.DeleteChat)
	e.PUT("/api/chat", ch.EditChat)
	e.PUT("/api/chat/:id/leave", ch.LeaveChat)
	e.PUT("/api/chat/:id/join", ch.JoinChat)
	e.GET("/api/chat/:id/participant/search", ch.GetParticipants)
	e.POST("/api/chat/:id/participant/filter", ch.FilterParticipants)
	e.POST("/api/chat/:id/participant/count", ch.CountParticipants)
	e.PUT("/api/chat/:id/participant", ch.AddParticipants)
	e.PUT("/api/chat/:id/participant/:participantId", ch.ChangeParticipant)
	e.DELETE("/api/chat/:id/participant/:participantId", ch.DeleteParticipant)
	e.GET("/api/chat/:id/user-candidate", ch.SearchForUsersToAdd)
	e.DELETE("/internal/delete-all-participants", ch.RemoveAllParticipants)
	e.GET("/internal/does-participant-belong-to-chat", ch.DoesParticipantBelongToChat)
	e.GET("/api/chat/:id/mention/suggest", ch.SearchForUsersToMention)
	e.GET("/api/chat/can-create-blog", ch.CanCreateBlog)
	e.PUT("/api/chat/tet-a-tet/:participantId", ch.TetATet)
	e.PUT("/api/chat/public/preview-without-html", ch.CreatePreview)
	e.GET("/internal/access", ch.CheckAccess)
	e.GET("/internal/participant-ids", ch.GetChatParticipants)
	e.GET("/internal/is-admin", ch.IsAdmin)
	e.GET("/internal/does-chats-exist", ch.IsExists)
	e.GET("/internal/name-for-invite", ch.GetNameForInvite)
	e.GET("/internal/basic/:id", ch.GetBasicInfo)

	e.PUT("/api/chat/:id/notification", ch.PutUserChatNotificationSettings)
	e.GET("/api/chat/:id/notification", ch.GetUserChatNotificationSettings)

	e.GET("/api/chat/:id/message/search", mc.GetMessages)
	e.GET("/api/chat/:id/message/:messageId", mc.GetMessage)
	e.POST("/api/chat/:id/message/fresh", mc.IsFreshMessagesPage)
	e.PUT("/api/chat/:id/message/:messageId/reaction", mc.ReactionMessage)
	e.POST("/api/chat/:id/message", mc.PostMessage)
	e.PUT("/api/chat/:id/message", mc.EditMessage)
	e.POST("/api/chat/:id/message/filter", mc.Filter)
	e.PUT("/api/chat/:id/message/file-item-uuid", mc.SetFileItemUuid)
	e.DELETE("/api/chat/:id/message/:messageId", mc.DeleteMessage)
	e.PUT("/api/chat/:id/message/read/:messageId", mc.ReadMessage)
	e.GET("/api/chat/:id/message/read/:messageId", mc.GetReadMessageUsers)
	e.GET("/api/chat/:id/message/find-by-file-item-uuid/:fileItemUuid", mc.FindMessageByFileItemUuid)

	e.PUT("/api/chat/:id/typing", mc.TypeMessage)
	e.PUT("/api/chat/:id/broadcast", mc.BroadcastMessage)
	e.DELETE("/internal/remove-file-item", mc.RemoveFileItem)

	e.GET("/api/chat/:id/message/pin", mc.GetPinnedMessages)
	e.GET("/api/chat/:id/message/pin/promoted", mc.GetPinnedPromotedMessage)
	e.PUT("/api/chat/:id/message/:messageId/pin", mc.PinMessage)
	e.PUT("/api/chat/:id/pin", ch.PinChat)
	e.PUT("/api/chat/:id/message/:messageId/publish", mc.PublishMessage)
	e.GET("/api/chat/:id/message/publish", mc.GetPublishedMessages)
	e.GET("/api/chat/public/:id/message/:messageId", mc.GetPublishedMessage)

	e.PUT("/api/chat/:id/read", ch.MarkAsRead)
	e.PUT("/api/chat/read", ch.MarkAsReadAll)

	e.PUT("/api/chat/:id/message/:messageId/blog-post", mc.MakeBlogPost)
	e.GET("/api/blog", bh.GetBlogPosts)
	e.GET("/internal/blog/seo", bh.GetAllBlogPostsForSeo)
	e.GET("/api/blog/:id", bh.GetBlogPost)
	e.GET("/api/blog/:id/comment", bh.GetBlogPostComments)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// do some work on application stop (like closing connections and files)
			lgr.Infof("Stopping http server")
			return e.Shutdown(ctx)
		},
	})

	return e
}

func configureTracer(lgr *logger.Logger, lc fx.Lifecycle) (*sdktrace.TracerProvider, error) {
	lgr.Infof("Configuring Jaeger tracing")

	conn, err := grpc.DialContext(context.Background(), viper.GetString("otlp.endpoint"), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
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

func runMigrations(db *db.DB, migrationsConfig *db.MigrationsConfig) {
	db.Migrate(migrationsConfig)
}

// rely on viper import and it's configured by
func runEcho(lgr *logger.Logger, e *echo.Echo) {
	address := viper.GetString("server.address")

	lgr.Info("Starting server...")
	// Start server in another goroutine
	go func() {
		if err := e.Start(address); err != nil {
			lgr.Infof("server shut down: %v", err)
		}
	}()
	lgr.Info("Server started. Waiting for interrupt signal 2 (Ctrl+C)")
}

func runScheduler(
	lgr *logger.Logger,
	scheduler *dcron.Cron,
	ct *tasks.CleanChatsOfDeletedUserTask,
	lc fx.Lifecycle,
) error {
	scheduler.Start()
	lgr.Infof("Scheduler started")

	if viper.GetBool("schedulers." + ct.Key() + ".enabled") {
		lgr.Infof("Adding task " + ct.Key() + " to scheduler")
		err := scheduler.AddJobs(ct)
		if err != nil {
			return err
		}
	} else {
		lgr.Infof("Task " + ct.Key() + " is disabled")
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
