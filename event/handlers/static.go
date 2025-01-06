package handlers

import (
	"embed"
	"github.com/labstack/echo/v4"
	"io/fs"
	"net/http"
	"nkonev.name/event/logger"
	"strings"
)

//go:embed static
var embeddedFiles embed.FS

type StaticMiddleware echo.MiddlewareFunc

func ConfigureStaticMiddleware(lgr *logger.Logger) StaticMiddleware {
	fsys, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		lgr.Panicf("Cannot open static embedded dir")
	}
	staticDir := http.FS(fsys)

	h := http.FileServer(staticDir)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqUrl := c.Request().RequestURI
			if reqUrl == "/" || reqUrl == "/index.html" || reqUrl == "/favicon.ico" || strings.HasPrefix(reqUrl, "/build") || strings.HasPrefix(reqUrl, "/assets") || reqUrl == "/git.json" {
				h.ServeHTTP(c.Response().Writer, c.Request())
				return nil
			} else {
				return next(c)
			}
		}
	}
}
