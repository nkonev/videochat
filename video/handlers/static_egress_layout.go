package handlers

import (
	"embed"
	"github.com/labstack/echo/v4"
	"io/fs"
	"net/http"
	. "nkonev.name/video/logger"
)

//go:embed static-egress-layout
var egressLayoutEmbeddedFiles embed.FS

type EgressLayoutStaticMiddleware echo.MiddlewareFunc

func ConfigureEgressLayoutStaticMiddleware() EgressLayoutStaticMiddleware {
	fsys, err := fs.Sub(egressLayoutEmbeddedFiles, "static-egress-layout")
	if err != nil {
		Logger.Panicf("Cannot open static embedded dir")
	}
	staticDir := http.FS(fsys)

	h := http.FileServer(staticDir)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h.ServeHTTP(c.Response().Writer, c.Request())
			return nil
		}
	}
}
