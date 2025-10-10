package handlers

import (
	"net/http"
	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/logger"

	"github.com/gin-gonic/gin"
)

type TechnicalHandler struct {
	lgr              *logger.LoggerWrapper
	eventBus         *cqrs.KafkaProducer
	dbWrapper        *db.DB
	commonProjection *cqrs.CommonProjection
	cfg              *config.AppConfig
}

func NewTechnicalHandler(
	lgr *logger.LoggerWrapper,
	eventBus *cqrs.KafkaProducer,
	dbWrapper *db.DB,
	commonProjection *cqrs.CommonProjection,
	cfg *config.AppConfig,
) *TechnicalHandler {
	return &TechnicalHandler{
		lgr:              lgr,
		eventBus:         eventBus,
		dbWrapper:        dbWrapper,
		commonProjection: commonProjection,
		cfg:              cfg,
	}
}

func (ch *TechnicalHandler) Health(g *gin.Context) {
	g.Status(http.StatusOK)
}

func (ch *TechnicalHandler) Truncate(g *gin.Context) {
	cc := cqrs.Truncate{}

	err := cc.Handle(g.Request.Context(), ch.eventBus, ch.dbWrapper, ch.commonProjection, ch.lgr, ch.cfg)
	if err != nil {
		if translateParticipantError(g, err) {
			return
		}

		ch.lgr.ErrorContext(g.Request.Context(), "Error sending ParticipantDelete command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}
