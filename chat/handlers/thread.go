package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/sanitizer"
	"nkonev.name/chat/utils"
)

type ThreadHandler struct {
	lgr                 *logger.LoggerWrapper
	eventBus            *cqrs.KafkaProducer
	dbWrapper           *db.DB
	commonProjection    *cqrs.CommonProjection
	stripTagsPolicy     *sanitizer.StripTagsPolicy
	enrichingProjection *cqrs.EnrichingProjection
	cfg                 *config.AppConfig
}

func NewThreadHandler(
	lgr *logger.LoggerWrapper,
	eventBus *cqrs.KafkaProducer,
	dbWrapper *db.DB,
	commonProjection *cqrs.CommonProjection,
	stripTagsPolicy *sanitizer.StripTagsPolicy,
	enrichingProjection *cqrs.EnrichingProjection,
	cfg *config.AppConfig,
) *ThreadHandler {
	return &ThreadHandler{
		lgr:                 lgr,
		eventBus:            eventBus,
		dbWrapper:           dbWrapper,
		commonProjection:    commonProjection,
		stripTagsPolicy:     stripTagsPolicy,
		enrichingProjection: enrichingProjection,
		cfg:                 cfg,
	}
}

func (ch *ThreadHandler) CreateThread(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mid := g.Param(dto.MessageIdParam)

	messageId, err := utils.ParseInt64(mid)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding messageId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cc := cqrs.ThreadCreate{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ChatId:         chatId,
		MessageId:      messageId,
	}

	threadId, err := cc.Handle(g.Request.Context(), ch.eventBus, ch.dbWrapper, ch.commonProjection)
	if err != nil {
		if translateChatError(g, err) {
			return
		}

		ch.lgr.ErrorContext(g.Request.Context(), "Error sending ThreadCreate command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	ch.lgr.InfoContext(g.Request.Context(), "created the thread", logger.AttributeChatId, chatId, logger.AttributeThreadId, threadId)

	m := dto.IdResponse{Id: threadId}

	g.JSON(http.StatusOK, m)
}

func (ch *ThreadHandler) DeleteThread(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mid := g.Param(dto.MessageIdParam)

	messageId, err := utils.ParseInt64(mid)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding messageId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cc := cqrs.ThreadDelete{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ChatId:         chatId,
		MessageId:      messageId,
	}

	err = cc.Handle(g.Request.Context(), ch.eventBus, ch.dbWrapper, ch.commonProjection)
	if err != nil {
		if translateChatError(g, err) {
			return
		}

		ch.lgr.ErrorContext(g.Request.Context(), "Error sending ThreadDelete command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	ch.lgr.InfoContext(g.Request.Context(), "created the thread", logger.AttributeChatId, chatId)

	g.Status(http.StatusOK)
}
