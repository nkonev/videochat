package main

import (
	"context"
	"contrib.go.opencensus.io/exporter/jaeger"
	"github.com/GeertJohan/go.rice"
	"github.com/centrifugal/centrifuge"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/microcosm-cc/bluemonday"
	uberCompat "github.com/nkonev/jaeger-uber-propagation-compat/propagation"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"go.uber.org/fx"
	"net/http"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/notifications"
	"nkonev.name/chat/utils"
	"strings"
)

type staticMiddleware echo.MiddlewareFunc

func main() {
	configFile := utils.InitFlags("./chat/config-dev/config.yml")
	utils.InitViper(configFile, "CHAT")

	app := fx.New(
		fx.Logger(Logger),
		fx.Provide(
			client.NewRestClient,
			handlers.ConfigureCentrifuge,
			handlers.CreateSanitizer,
			configureEcho,
			configureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			db.ConfigureDb,
			notifications.NewNotifications,
		),
		fx.Invoke(
			initJaeger,
			runMigrations,
			runCentrifuge,
			runEcho,
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

func configureOpencensusMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) (err error) {
			handler := &ochttp.Handler{
				Handler: http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						ctx.SetRequest(r)
						ctx.SetResponse(echo.NewResponse(w, ctx.Echo()))
						err = next(ctx)
					},
				),
				Propagation: &uberCompat.HTTPFormat{},
			}
			handler.ServeHTTP(ctx.Response(), ctx.Request())
			return
		}
	}
}

func configureEcho(
	staticMiddleware staticMiddleware,
	authMiddleware handlers.AuthMiddleware,
	lc fx.Lifecycle,
	notificator notifications.Notifications,
	node *centrifuge.Node,
	db db.DB,
	policy *bluemonday.Policy,
	restClient client.RestClient,
) *echo.Echo {

	bodyLimit := viper.GetString("server.body.limit")

	e := echo.New()
	e.Logger.SetOutput(Logger.Writer())

	e.Pre(echo.MiddlewareFunc(staticMiddleware))
	e.Use(configureOpencensusMiddleware())
	e.Use(echo.MiddlewareFunc(authMiddleware))
	accessLoggerConfig := middleware.LoggerConfig{
		Output: Logger.Writer(),
		Format: `"remote_ip":"${remote_ip}",` +
			`"method":"${method}","uri":"${uri}",` +
			`"status":${status},` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out},"traceId":"${header:X-B3-Traceid}"` + "\n",
	}
	e.Use(middleware.LoggerWithConfig(accessLoggerConfig))
	e.Use(middleware.Secure())
	e.Use(middleware.BodyLimit(bodyLimit))

	e.GET("/chat/websocket", handlers.Convert(handlers.CentrifugeAuthMiddleware(centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{}))))

	e.GET("/chat", handlers.GetChats(db, restClient))
	e.GET("/chat/:id", handlers.GetChat(db, restClient))
	e.POST("/chat", handlers.CreateChat(db, notificator, restClient))
	e.DELETE("/chat/:id", handlers.DeleteChat(db))
	e.PUT("/chat", handlers.EditChat(db, notificator, restClient))

	e.GET("/chat/:id/message", handlers.GetMessages(db))
	e.GET("/chat/:id/message/:messageId", handlers.GetMessage(db))
	e.POST("/chat/:id/message", handlers.PostMessage(db, policy, notificator))
	e.PUT("/chat/:id/message", handlers.EditMessage(db, policy))
	e.DELETE("/chat/:id/message/:messageId", handlers.DeleteMessage(db))

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// do some work on application stop (like closing connections and files)
			Logger.Infof("Stopping http server")
			return e.Shutdown(ctx)
		},
	})

	return e
}

func configureStaticMiddleware() staticMiddleware {
	box := rice.MustFindBox("static").HTTPBox()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqUrl := c.Request().RequestURI
			if reqUrl == "/" || reqUrl == "/index.html" || reqUrl == "/favicon.ico" || strings.HasPrefix(reqUrl, "/build") || strings.HasPrefix(reqUrl, "/assets") {
				http.FileServer(box).
					ServeHTTP(c.Response().Writer, c.Request())
				return nil
			} else {
				return next(c)
			}
		}
	}
}

func initJaeger(lc fx.Lifecycle) error {
	exporter, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: viper.GetString("jaeger.endpoint"),
		Process: jaeger.Process{
			ServiceName: "chat",
		},
	})
	if err != nil {
		return err
	}
	trace.RegisterExporter(exporter)
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			Logger.Infof("Stopping tracer")
			exporter.Flush()
			trace.UnregisterExporter(exporter)
			return nil
		},
	})
	return nil
}

func runMigrations(db db.DB) {
	db.Migrate()
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
