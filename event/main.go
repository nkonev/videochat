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
	"nkonev.name/event/client"
	"nkonev.name/event/config"
	"nkonev.name/event/graph"
	"nkonev.name/event/graph/generated"
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

	app := fx.New(
		fx.Logger(Logger),
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
		GetLogEntry(c.Request().Context()).Errorf("Unhandled error: %v", err)
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

	e.Any(GRAPHQL_PATH, handlers.Convert(graphQlServer))
	e.GET(GRAPHQL_PLAYGROUND, handlers.Convert(graphQlPlayground))

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// do some work on application stop (like closing connections and files)
			Logger.Infof("Stopping http server")
			return e.Shutdown(ctx)
		},
	})

	return e
}

func configureGraphQlServer(bus *eventbus.Bus, httpClient *client.RestClient) *handler.Server {
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{bus, httpClient}}))
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
	return srv
}

type GraphQlPlayground struct {
	http.HandlerFunc
}

func configureGraphQlPlayground() *GraphQlPlayground {
	return &GraphQlPlayground{playground.Handler("GraphQL playground", GRAPHQL_PATH)}
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

func configureEventBus(lc fx.Lifecycle) *eventbus.Bus {
	b := eventbus.New()
	Logger.Infof("Starting event bus")
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			Logger.Infof("Stopping event bus")
			b.Close()
			return nil
		},
	})
	return b
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
