package main

import (
	"context"
	"github.com/centrifugal/centrifuge"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
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
			handlers.ConfigureCentrifuge,
			handlers.CreateSanitizer,
			handlers.NewChatHandler,
			handlers.NewMessageHandler,
			configureEcho,
			handlers.ConfigureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			configureMigrations,
			db.ConfigureDb,
			services.NewNotifications,
			producer.NewRabbitNotificationsPublisher,
			listener.CreateAaaUserProfileUpdateListener,
			listener.CreateVideoInviteListener,
			listener.CreateVideoDialStatusListener,
			rabbitmq.CreateRabbitMqConnection,
			listener.CreateAaaChannel,
			listener.CreateVideoInviteChannel,
			listener.CreateVideoDialStatusChannel,
			listener.CreateAaaQueue,
			listener.CreateVideoInviteQueue,
			listener.CreateVideoDialStatusQueue,
		),
		fx.Invoke(
			runMigrations,
			runCentrifuge,
			runEcho,
			listener.ListenAaaQueue,
			listener.ListenVideoInviteQueue,
			listener.ListenVideoDialStatusQueue,
		),
	)
	app.Run()

	Logger.Infof("Exit program")
}

func runCentrifuge(node *centrifuge.Node) {
	// Run node.
	Logger.Infof("Starting centrifuge...")
	go func() {
		if err := node.Run(); err != nil {
			Logger.Fatalf("Error on start centrifuge: %v", err)
		}
	}()
	Logger.Info("Centrifuge started.")
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
		GetLogEntry(c.Request().Context()).Errorf("Unhandled error: %v", err)
		originalHandler(err, c)
	}
}

func configureEcho(
	staticMiddleware handlers.StaticMiddleware,
	authMiddleware handlers.AuthMiddleware,
	lc fx.Lifecycle,
	node *centrifuge.Node,
	ch *handlers.ChatHandler,
	mc *handlers.MessageHandler,
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

	e.GET("/chat/websocket", handlers.Convert(handlers.CentrifugeAuthMiddleware(centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{}))))

	e.GET("/chat", ch.GetChats)
	e.GET("/chat/:id", ch.GetChat)
	e.POST("/chat", ch.CreateChat)
	e.DELETE("/chat/:id", ch.DeleteChat)
	e.PUT("/chat", ch.EditChat)
	e.PUT("/chat/:id/leave", ch.LeaveChat)
	e.PUT("/chat/:id/user/:participantId", ch.ChangeParticipant)
	e.DELETE("/chat/:id/user/:participantId", ch.DeleteParticipant)
	e.DELETE("/internal/delete-all-participants", ch.RemoveAllParticipants)
	e.GET("/internal/does-participant-belong-to-chat", ch.DoesParticipantBelongToChat)
	e.PUT("/chat/:id/users", ch.AddParticipants)
	e.GET("/chat/:id/user", ch.SearchForUsersToAdd)
	e.PUT("/chat/tet-a-tet/:participantId", ch.TetATet)
	e.GET("/internal/access", ch.CheckAccess)
	e.GET("/internal/participant-ids", ch.GetChatParticipants)
	e.GET("/internal/is-admin", ch.IsAdmin)
	e.GET("/internal/is-chat-exists/:id", ch.IsExists)

	e.GET("/chat/:id/message", mc.GetMessages)
	e.GET("/chat/:id/message/:messageId", mc.GetMessage)
	e.POST("/chat/:id/message", mc.PostMessage)
	e.PUT("/chat/:id/message", mc.EditMessage)
	e.DELETE("/chat/:id/message/:messageId", mc.DeleteMessage)
	e.PUT("/chat/:id/typing", mc.TypeMessage)
	e.PUT("/chat/:id/broadcast", mc.BroadcastMessage)
	e.DELETE("/internal/remove-file-item", mc.RemoveFileItem)
	e.POST("/internal/check-embedded-files", mc.CheckEmbeddedFiles)

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

func configureMigrations() db.MigrationsConfig {
	return db.MigrationsConfig{}
}

func runMigrations(db db.DB, migrationsConfig db.MigrationsConfig) {
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
