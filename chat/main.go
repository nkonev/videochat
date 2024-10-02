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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"nkonev.name/chat/client"
	"nkonev.name/chat/config"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers"
	"nkonev.name/chat/listener"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/services"
	"nkonev.name/chat/tasks"
	"nkonev.name/chat/type_registry"
)

const EXTERNAL_TRACE_ID_HEADER = "trace-id"
const TRACE_RESOURCE = "chat"

func main() {
	config.InitViper()

	app := fx.New(
		fx.Logger(Logger),
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

func configureEcho(
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

	e.GET("/api/chat", ch.GetChats)
	e.GET("/api/chat/has-new-messages", ch.HasNewMessages)
	e.POST("/api/chat/filter", ch.Filter)
	e.GET("/api/chat/:id", ch.GetChat)
	e.POST("/api/chat/fresh", ch.IsFreshChatsPage)
	e.POST("/api/chat", ch.CreateChat)
	e.DELETE("/api/chat/:id", ch.DeleteChat)
	e.PUT("/api/chat", ch.EditChat)
	e.PUT("/api/chat/:id/leave", ch.LeaveChat)
	e.PUT("/api/chat/:id/join", ch.JoinChat)
	e.GET("/api/chat/:id/participant", ch.GetParticipants)
	e.POST("/api/chat/:id/participant/filter", ch.FilterParticipants)
	e.POST("/api/chat/:id/participant/count", ch.CountParticipants)
	e.PUT("/api/chat/:id/participant", ch.AddParticipants)
	e.PUT("/api/chat/:id/participant/:participantId", ch.ChangeParticipant)
	e.DELETE("/api/chat/:id/participant/:participantId", ch.DeleteParticipant)
	e.GET("/api/chat/:id/user-candidate", ch.SearchForUsersToAdd)
	e.DELETE("/internal/delete-all-participants", ch.RemoveAllParticipants)
	e.GET("/internal/does-participant-belong-to-chat", ch.DoesParticipantBelongToChat)
	e.GET("/api/chat/:id/suggest-participants", ch.SearchForUsersToMention)
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

	e.GET("/api/chat/:id/message", mc.GetMessages)
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
			Logger.Infof("Stopping http server")
			return e.Shutdown(ctx)
		},
	})

	return e
}

func configureTracer(lc fx.Lifecycle) (*sdktrace.TracerProvider, error) {
	Logger.Infof("Configuring Jaeger tracing")

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
func runEcho(e *echo.Echo) {
	address := viper.GetString("server.address")

	Logger.Info("Starting server...")
	// Start server in another goroutine
	go func() {
		if err := e.Start(address); err != nil {
			Logger.Infof("server shut down: %v", err)
		}
	}()
	Logger.Info("Server started. Waiting for interrupt signal 2 (Ctrl+C)")
}

func runScheduler(
	scheduler *dcron.Cron,
	ct *tasks.CleanChatsOfDeletedUserTask,
	lc fx.Lifecycle,
) error {
	scheduler.Start()
	Logger.Infof("Scheduler started")

	if viper.GetBool("schedulers." + ct.Key() + ".enabled") {
		Logger.Infof("Adding " + ct.Key() + " job to scheduler")
		err := scheduler.AddJobs(ct)
		if err != nil {
			return err
		}
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			Logger.Infof("Stopping scheduler")
			<-scheduler.Stop().Done()
			return nil
		},
	})
	return nil
}
