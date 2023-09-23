package main

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	jaegerPropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	jaegerExporter "go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
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
			services.NewEvents,
			producer.NewRabbitEventsPublisher,
			producer.NewRabbitNotificationsPublisher,
			listener.CreateAaaUserProfileUpdateListener,
			rabbitmq.CreateRabbitMqConnection,
			listener.CreateAaaChannel,
			listener.CreateAaaQueue,
		),
		fx.Invoke(
			runMigrations,
			runEcho,
			listener.ListenAaaQueue,
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
					if existsSpan.SpanContext().HasSpanID() {
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
		errUnwrapped, ok := err.(tracerr.Error)
		if !ok {
			GetLogEntry(c.Request().Context()).Errorf("Unhandled and unwrappable error: %v", err)
		} else {
			GetLogEntry(c.Request().Context()).Errorf("Unhandled error: %v, stack: %v", errUnwrapped.Error(), errUnwrapped.StackTrace())
			tracerr.PrintSource(err)
		}
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

	e.GET("/chat", ch.GetChats)
	e.GET("/chat/:id", ch.GetChat)
	e.POST("/chat", ch.CreateChat)
	e.DELETE("/chat/:id", ch.DeleteChat)
	e.PUT("/chat", ch.EditChat)
	e.PUT("/chat/:id/leave", ch.LeaveChat)
	e.PUT("/chat/:id/join", ch.JoinChat)
	e.GET("/chat/:id/user", ch.GetParticipants)
	e.PUT("/chat/:id/user", ch.AddParticipants)
	e.PUT("/chat/:id/user/:participantId", ch.ChangeParticipant)
	e.DELETE("/chat/:id/user/:participantId", ch.DeleteParticipant)
	e.GET("/chat/:id/user-candidate", ch.SearchForUsersToAdd)
	e.DELETE("/internal/delete-all-participants", ch.RemoveAllParticipants)
	e.GET("/internal/does-participant-belong-to-chat", ch.DoesParticipantBelongToChat)
	e.GET("/chat/:id/suggest-participants", ch.SearchForUsersToMention)
	e.GET("/chat/get-page", ch.GetChatPage)
	e.PUT("/chat/tet-a-tet/:participantId", ch.TetATet)
	e.PUT("/chat/public/preview-without-html", ch.CreatePreview)
	e.GET("/internal/access", ch.CheckAccess)
	e.GET("/internal/participant-ids", ch.GetChatParticipants)
	e.GET("/internal/is-admin", ch.IsAdmin)
	e.GET("/internal/is-chat-exists/:id", ch.IsExists)
	e.GET("/internal/name-for-invite", ch.GetNameForInvite)
	e.GET("/internal/basic/:id", ch.GetBasicInfo)

	e.GET("/chat/:id/message", mc.GetMessages)
	e.GET("/chat/:id/message/:messageId", mc.GetMessage)
	e.POST("/chat/:id/message", mc.PostMessage)
	e.PUT("/chat/:id/message", mc.EditMessage)
	e.PUT("/chat/:id/message/file-item-uuid", mc.SetFileItemUuid)
	e.DELETE("/chat/:id/message/:messageId", mc.DeleteMessage)
	e.PUT("/chat/:id/message/read/:messageId", mc.ReadMessage)
	e.GET("/chat/:id/message/read/:messageId", mc.GetReadMessageUsers)
	e.PUT("/chat/:id/typing", mc.TypeMessage)
	e.PUT("/chat/:id/broadcast", mc.BroadcastMessage)
	e.DELETE("/internal/remove-file-item", mc.RemoveFileItem)

	e.GET("/chat/:id/message/pin", mc.GetPinnedMessages)
	e.GET("/chat/:id/message/pin/promoted", mc.GetPinnedPromotedMessage)
	e.PUT("/chat/:id/message/:messageId/pin", mc.PinMessage)
	e.PUT("/chat/:id/pin", ch.PinChat)

	e.PUT("/chat/:id/message/:messageId/blog-post", mc.MakeBlogPost)
	e.GET("/blog", bh.GetBlogPosts)
	e.GET("/blog/:id", bh.GetBlogPost)
	e.GET("/blog/:id/comment", bh.GetBlogPostComments)

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
	endpoint := jaegerExporter.WithAgentEndpoint(
		jaegerExporter.WithAgentHost(viper.GetString("jaeger.host")),
		jaegerExporter.WithAgentPort(viper.GetString("jaeger.port")),
	)
	exporter, err := jaegerExporter.New(endpoint)
	if err != nil {
		return nil, err
	}
	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(TRACE_RESOURCE),
	)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resources),
	)
	otel.SetTracerProvider(tp)
	jaeger := jaegerPropagator.Jaeger{}
	// register jaeger propagator
	otel.SetTextMapPropagator(jaeger)
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
