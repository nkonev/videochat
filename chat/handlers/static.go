package handlers

import (
	"embed"
	"errors"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"
)

type StaticHandler struct {
	lgr       *logger.LoggerWrapper
	cfg       *config.AppConfig
	staticDir http.FileSystem
}

//go:embed static
var embeddedFiles embed.FS

func NewStaticHandler(lgr *logger.LoggerWrapper, cfg *config.AppConfig) (*StaticHandler, error) {
	fsys, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		return nil, errors.New("Cannot open static embedded dir")
	}
	staticDir := http.FS(fsys)

	return &StaticHandler{lgr: lgr, cfg: cfg, staticDir: staticDir}, nil
}

func (mc *StaticHandler) StaticGitJson(g *gin.Context) {
	g.FileFromFS("/"+gitJson, mc.staticDir)
}
