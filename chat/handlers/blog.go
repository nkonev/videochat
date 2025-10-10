package handlers

import (
	"net/http"
	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"

	"github.com/gin-gonic/gin"
)

type BlogHandler struct {
	lgr                 *logger.LoggerWrapper
	eventBus            *cqrs.KafkaProducer
	dbWrapper           *db.DB
	commonProjection    *cqrs.CommonProjection
	enrichingProjection *cqrs.EnrichingProjection
	cfg                 *config.AppConfig
}

func NewBlogHandler(
	lgr *logger.LoggerWrapper,
	eventBus *cqrs.KafkaProducer,
	dbWrapper *db.DB,
	commonProjection *cqrs.CommonProjection,
	enrichingProjection *cqrs.EnrichingProjection,
	cfg *config.AppConfig,
) *BlogHandler {
	return &BlogHandler{
		lgr:                 lgr,
		eventBus:            eventBus,
		dbWrapper:           dbWrapper,
		commonProjection:    commonProjection,
		enrichingProjection: enrichingProjection,
		cfg:                 cfg,
	}
}

func (ch *BlogHandler) SearchBlogs(g *gin.Context) {
	page := utils.FixPageString(g.Query(dto.PageParam))
	size := utils.FixSizeString(g.Query(dto.SizeParam))
	offset := utils.GetOffset(page, size)
	reverse := utils.GetBooleanOr(g.Query(dto.ReverseParam), false)
	searchString := g.Query(dto.SearchStringParam)

	blogs, err := ch.enrichingProjection.GetBlogsEnriched(g.Request.Context(), size, offset, cqrs.BlogOrderByCreateDateTime, reverse, searchString)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error getting blogs", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, blogs)
}

func (ch *BlogHandler) GetBlog(g *gin.Context) {
	cid := g.Param(dto.BlogIdParam)

	blogId, err := utils.ParseInt64(cid)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding blogId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	blog, err := ch.enrichingProjection.GetBlogEnriched(g.Request.Context(), blogId)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error getting blog", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	if blog == nil {
		g.Status(http.StatusNoContent)
		return
	}

	g.JSON(http.StatusOK, blog)
}

func (ch *BlogHandler) SearchComments(g *gin.Context) {
	cid := g.Param(dto.BlogIdParam)
	blogId, err := utils.ParseInt64(cid)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding blogId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	page := utils.FixPageString(g.Query(dto.PageParam))
	size := utils.FixSizeString(g.Query(dto.SizeParam))
	offset := utils.GetOffset(page, size)
	reverse := utils.GetBooleanOr(g.Query(dto.ReverseParam), false)

	comments, err := ch.enrichingProjection.GetCommentsEnriched(g.Request.Context(), blogId, size, offset, reverse)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error getting blog comments", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, comments)
}

func (ch *BlogHandler) CanCreateBlog(g *gin.Context) {
	userPermissions := getUserPermissions(g)

	can := cqrs.IsBloggingAllowed(ch.cfg, userPermissions)
	g.JSON(http.StatusOK, &utils.H{"canCreateBlog": can})
}

func (ch *BlogHandler) GetAllBlogPostsForSeo(g *gin.Context) {
	page := utils.FixPageString(g.Query(dto.PageParam))
	size := utils.FixSizeString(g.Query(dto.SizeParam))
	offset := utils.GetOffset(page, size)

	blogs, err := ch.enrichingProjection.GetBlogsEnrichedForSeo(g.Request.Context(), size, offset)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error getting blogs", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, blogs)
}
