package main

import (
	"context"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/montag451/go-eventbus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	gqlgen_opentelemetry "github.com/zhevron/gqlgen-opentelemetry/v2"
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
	"nkonev.name/event/client"
	"nkonev.name/event/config"
	"nkonev.name/event/graph"
	"nkonev.name/event/handlers"
	"nkonev.name/event/listener"
	. "nkonev.name/event/logger"
	"nkonev.name/event/rabbitmq"
	"nkonev.name/event/type_registry"
	"time"
)

const EXTERNAL_TRACE_ID_HEADER = "trace-id"
const TRACE_RESOURCE = "event"
const GRAPHQL_PATH = "/api/event/graphql"
const GRAPHQL_PLAYGROUND = "/event/playground"

func main() {
	config.InitViper()
	lgr := NewLogger()

	app := fx.New(
		fx.Logger(lgr),
		fx.Supply(lgr),
		fx.Provide(
			configureTracer,
			configureGraphQlServer,
			configureGraphQlPlayground,
			configureEcho,
			configureEventBus,
			handlers.ConfigureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			listener.CreateEventsListener,
			rabbitmq.CreateRabbitMqConnection,
			type_registry.NewTypeRegistryInstance,
			client.NewRestClient,
		),
		fx.Invoke(
			runEcho,
			listener.CreateEventsChannel,
			listener.CreateAaaChannel,
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
		GetLogEntry(c.Request().Context(), lgr).Errorf("Unhandled error: %v", err)
		originalHandler(err, c)
	}
}

func configureEcho(
	staticMiddleware handlers.StaticMiddleware,
	authMiddleware handlers.AuthMiddleware,
	lc fx.Lifecycle,
	tp *sdktrace.TracerProvider,
	graphQlServer *handler.Server,
	graphQlPlayground *GraphQlPlayground,
	lgr *log.Logger,
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

	e.Any(GRAPHQL_PATH, handlers.Convert(graphQlServer))
	e.GET(GRAPHQL_PLAYGROUND, handlers.Convert(graphQlPlayground))

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// do some work on application stop (like closing connections and files)
			lgr.Infof("Stopping http server")
			return e.Shutdown(ctx)
		},
	})

	return e
}

func configureGraphQlServer(lgr *log.Logger, bus *eventbus.Bus, httpClient *client.RestClient, tp *sdktrace.TracerProvider) *handler.Server {
	tr := otel.Tracer("graphql")
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{bus, httpClient, tr, lgr}}))
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	})
	srv.Use(extension.Introspection{})
	srv.Use(gqlgen_opentelemetry.Tracer{
		TracerProvider: tp,
	})
	return srv
}

type GraphQlPlayground struct {
	http.HandlerFunc
}

func configureGraphQlPlayground() *GraphQlPlayground {
	return &GraphQlPlayground{playground.Handler("GraphQL playground", GRAPHQL_PATH)}
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

func configureEventBus(lgr *log.Logger, lc fx.Lifecycle) *eventbus.Bus {
	b := eventbus.New()
	lgr.Infof("Starting event bus")
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Infof("Stopping event bus")
			b.Close()
			return nil
		},
	})
	return b
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
