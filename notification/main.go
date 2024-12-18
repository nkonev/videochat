package main

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rotisserie/eris"
	log "github.com/sirupsen/logrus"
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
	"nkonev.name/notification/config"
	"nkonev.name/notification/db"
	"nkonev.name/notification/handlers"
	"nkonev.name/notification/listener"
	. "nkonev.name/notification/logger"
	"nkonev.name/notification/producer"
	"nkonev.name/notification/rabbitmq"
	"nkonev.name/notification/services"
)

const EXTERNAL_TRACE_ID_HEADER = "trace-id"
const TRACE_RESOURCE = "notification"

func main() {
	config.InitViper()
	lgr := NewLogger()

	app := fx.New(
		fx.Logger(lgr),
		fx.Supply(lgr),
		fx.Provide(
			configureTracer,
			configureEcho,
			handlers.ConfigureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			handlers.NewMessageHandler,
			configureMigrations,
			db.ConfigureDb,
			listener.CreateNotificationsListener,
			rabbitmq.CreateRabbitMqConnection,
			services.CreateNotificationService,
			producer.NewRabbiEventPublisher,
		),
		fx.Invoke(
			runMigrations,
			runEcho,
			listener.CreateNotificationsChannel,
		),
	)
	app.Run()

	lgr.Infof("Exit program")
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

func createCustomHTTPErrorHandler(lgr *log.Logger, e *echo.Echo) func(err error, c echo.Context) {
	originalHandler := e.DefaultHTTPErrorHandler
	return func(err error, c echo.Context) {
		formattedStr := eris.ToString(err, true)
		GetLogEntry(c.Request().Context(), lgr).Errorf("Unhandled error: %v", formattedStr)
		originalHandler(err, c)
	}
}

func configureEcho(
	lgr *log.Logger,
	staticMiddleware handlers.StaticMiddleware,
	authMiddleware handlers.AuthMiddleware,
	ch *handlers.NotificationHandler,
	lc fx.Lifecycle,
	tp *sdktrace.TracerProvider,
) *echo.Echo {

	bodyLimit := viper.GetString("server.body.limit")

	e := echo.New()
	e.Logger.SetOutput(lgr.Writer())

	e.HTTPErrorHandler = createCustomHTTPErrorHandler(lgr, e)

	e.Pre(echo.MiddlewareFunc(staticMiddleware))
	e.Use(configureOpentelemetryMiddleware(tp))
	e.Use(configureWriteHeaderMiddleware())
	e.Use(echo.MiddlewareFunc(authMiddleware))
	accessLoggerConfig := middleware.LoggerConfig{
		Output: lgr.Writer(),
		Format: `"remote_ip":"${remote_ip}",` +
			`"method":"${method}","uri":"${uri}",` +
			`"status":${status},` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out},"traceId":"${header:uber-trace-id}"` + "\n",
	}
	e.Use(middleware.LoggerWithConfig(accessLoggerConfig))
	e.Use(middleware.Secure())
	e.Use(middleware.BodyLimit(bodyLimit))

	e.GET("/api/notification/list", ch.GetNotifications)
	e.GET("/api/notification/count", ch.GetNotificationsCount)
	e.DELETE("/api/notification", ch.DeleteAllNotifications)
	e.GET("/api/notification/settings/global", ch.GetGlobalNotificationSettings)
	e.PUT("/api/notification/settings/global", ch.PutGlobalNotificationSettings)
	e.GET("/api/notification/settings/:id/chat", ch.GetChatNotificationSettings)
	e.PUT("/api/notification/settings/:id/chat", ch.PutChatNotificationSettings)
	e.PUT("/api/notification/read/:notificationId", ch.ReadNotification)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// do some work on application stop (like closing connections and files)
			lgr.Infof("Stopping http server")
			return e.Shutdown(ctx)
		},
	})

	return e
}

func configureTracer(lgr *log.Logger, lc fx.Lifecycle) (*sdktrace.TracerProvider, error) {
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
				lgr.Printf("Error shutting down tracer provider: %v", err)
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
func runEcho(lgr *log.Logger, e *echo.Echo) {
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
