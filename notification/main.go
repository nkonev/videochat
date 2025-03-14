package main

import (
	"context"
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
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"nkonev.name/notification/app"
	"nkonev.name/notification/config"
	"nkonev.name/notification/db"
	"nkonev.name/notification/handlers"
	"nkonev.name/notification/listener"
	"nkonev.name/notification/logger"
	"nkonev.name/notification/producer"
	"nkonev.name/notification/rabbitmq"
	"nkonev.name/notification/services"
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
			configureEcho,
			handlers.ConfigureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			handlers.NewMessageHandler,
			configureMigrations,
			db.ConfigureDb,
			listener.CreateNotificationsPersistentListener,
			listener.CreateNotificationsEphemeralListener,
			rabbitmq.CreateRabbitMqConnection,
			services.CreateNotificationPersistentService,
			services.CreateNotificationEphemeralService,
			producer.NewRabbiEventPublisher,
			producer.NewRabbitInvitePublisher,
		),
		fx.Invoke(
			runMigrations,
			runEcho,
			listener.CreateNotificationsPersistentChannel,
			listener.CreateNotificationsEphemeralChannel,
		),
	)
	appFx.Run()

	lgr.Infof("Exit program")
	lgr.CloseLogger()
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

func createCustomHTTPErrorHandler(lgr *logger.Logger, e *echo.Echo) func(err error, c echo.Context) {
	originalHandler := e.DefaultHTTPErrorHandler
	return func(err error, c echo.Context) {
		formattedStr := eris.ToString(err, true)
		lgr.WithTracing(c.Request().Context()).Errorf("Unhandled error: %v", formattedStr)
		originalHandler(err, c)
	}
}

func configureEcho(
	lgr *logger.Logger,
	staticMiddleware handlers.StaticMiddleware,
	authMiddleware handlers.AuthMiddleware,
	ch *handlers.NotificationHandler,
	lc fx.Lifecycle,
	tp *sdktrace.TracerProvider,
) *echo.Echo {

	bodyLimit := viper.GetString("server.body.limit")

	e := echo.New()
	e.Logger.SetOutput(lgr)

	e.HTTPErrorHandler = createCustomHTTPErrorHandler(lgr, e)

	e.Pre(echo.MiddlewareFunc(staticMiddleware))
	e.Use(configureOpentelemetryMiddleware(tp))
	e.Use(configureWriteHeaderMiddleware())
	e.Use(echo.MiddlewareFunc(authMiddleware))

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
