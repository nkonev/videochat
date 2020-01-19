package main

import (
	"context"
	"github.com/GeertJohan/go.rice"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	. "github.com/nkonev/videochat/logger"
	"github.com/nkonev/videochat/utils"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/fx"
	"net/http"
	"strings"
)

type staticMiddleware echo.MiddlewareFunc

func main() {
	configFile := utils.InitFlag("./config-dev/config.yml")
	utils.InitViper(configFile)

	app := fx.New(
		fx.Logger(Logger),
		fx.Provide(
			configureMongo,
			configureEcho,
			configureStaticMiddleware,
		),
		fx.Invoke(runEcho),
	)
	app.Run()

	Logger.Infof("Exit program")
}

func configureEcho(staticMiddleware staticMiddleware, lc fx.Lifecycle) *echo.Echo {
	bodyLimit := viper.GetString("server.body.limit")

	e := echo.New()
	e.Logger.SetOutput(Logger.Writer())

	e.Pre(echo.MiddlewareFunc(staticMiddleware))

	accessLoggerConfig := middleware.LoggerConfig{
		Output: Logger.Writer(),
		Format: `"remote_ip":"${remote_ip}",` +
			`"method":"${method}","uri":"${uri}",` +
			`"status":${status},"error":"${error}","latency_human":"${latency_human}"` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out},"user_agent":"${user_agent}"` + "\n",
	}
	e.Use(middleware.LoggerWithConfig(accessLoggerConfig))
	e.Use(middleware.Secure())
	e.Use(middleware.BodyLimit(bodyLimit))

	//e.GET("/users", fsh.UsersHandler)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// do some work on application stop (like closing connections and files)
			Logger.Infof("Stopping server")
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
			if reqUrl == "/" || reqUrl == "/index.html" || reqUrl == "/favicon.ico" || strings.HasPrefix(reqUrl, "/build") || strings.HasPrefix(reqUrl, "/test-assets") {
				http.FileServer(box).
					ServeHTTP(c.Response().Writer, c.Request())
				return nil
			} else {
				return next(c)
			}
		}
	}
}

func configureMongo(lc fx.Lifecycle) *mongo.Client {
	mongoClient := utils.GetMongoClient()
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// do some work on application stop (like closing connections and files)
			Logger.Infof("Stopping mongo client")
			return mongoClient.Disconnect(ctx)
		},
	})

	return mongoClient
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
	Logger.Info("Server started. Waiting for interrupt (2) (Ctrl+C)")
}
