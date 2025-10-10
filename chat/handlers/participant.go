package handlers

import (
	"errors"
	"net/http"
	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"

	"github.com/gin-gonic/gin"
)

type ParticipantHandler struct {
	lgr                 *logger.LoggerWrapper
	eventBus            *cqrs.KafkaProducer
	dbWrapper           *db.DB
	commonProjection    *cqrs.CommonProjection
	enrichingProjection *cqrs.EnrichingProjection
	cfg                 *config.AppConfig
}

func NewParticipantHandler(
	lgr *logger.LoggerWrapper,
	eventBus *cqrs.KafkaProducer,
	dbWrapper *db.DB,
	commonProjection *cqrs.CommonProjection,
	enrichingProjection *cqrs.EnrichingProjection,
	cfg *config.AppConfig,
) *ParticipantHandler {
	return &ParticipantHandler{
		lgr:                 lgr,
		eventBus:            eventBus,
		dbWrapper:           dbWrapper,
		commonProjection:    commonProjection,
		enrichingProjection: enrichingProjection,
		cfg:                 cfg,
	}
}

func (ch *ParticipantHandler) AddParticipant(g *gin.Context) {
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

	ccd := new(dto.ParticipantAddDto)

	err = g.Bind(ccd)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding ParticipantAddDto", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cc := cqrs.ParticipantAdd{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ParticipantIds: ccd.ParticipantIds,
		ChatId:         chatId,
	}

	err = cc.Handle(g.Request.Context(), ch.eventBus, ch.dbWrapper, ch.commonProjection, ch.cfg)
	if err != nil {
		if translateParticipantError(g, err) {
			return
		}

		ch.lgr.ErrorContext(g.Request.Context(), "Error sending ParticipantAdd command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (ch *ParticipantHandler) DeleteParticipant(g *gin.Context) {
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

	interestingUserId, err := utils.ParseInt64(g.Param(dto.ParticipantIdParam))
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding participantId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cc := cqrs.ParticipantDelete{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ParticipantIds: []int64{interestingUserId},
		ChatId:         chatId,
	}

	err = cc.Handle(g.Request.Context(), ch.eventBus, ch.dbWrapper, ch.commonProjection, ch.cfg)
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

func (ch *ParticipantHandler) ChangeParticipant(g *gin.Context) {
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

	interestingUserId, err := utils.ParseInt64(g.Param(dto.ParticipantIdParam))
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding participantId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	newAdmin := utils.GetBoolean(g.Query(dto.AdminParam))

	cc := cqrs.ParticipantChange{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ParticipantId:  interestingUserId,
		ChatId:         chatId,
		NewAdmin:       newAdmin,
	}

	err = cc.Handle(g.Request.Context(), ch.eventBus, ch.dbWrapper, ch.commonProjection)
	if err != nil {
		if translateParticipantError(g, err) {
			return
		}

		ch.lgr.ErrorContext(g.Request.Context(), "Error sending ParticipantChange command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (ch *ParticipantHandler) LeaveChat(g *gin.Context) {
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

	cc := cqrs.ParticipantDelete{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ParticipantIds: []int64{userId},
		ChatId:         chatId,
		IsLeaving:      true,
	}

	err = cc.Handle(g.Request.Context(), ch.eventBus, ch.dbWrapper, ch.commonProjection, ch.cfg)
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

func (ch *ParticipantHandler) JoinChat(g *gin.Context) {
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

	cc := cqrs.ParticipantAdd{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ParticipantIds: []int64{userId},
		ChatId:         chatId,
		IsJoining:      true,
	}

	err = cc.Handle(g.Request.Context(), ch.eventBus, ch.dbWrapper, ch.commonProjection, ch.cfg)
	if err != nil {
		if translateParticipantError(g, err) {
			return
		}

		ch.lgr.ErrorContext(g.Request.Context(), "Error sending ParticipantAdd command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (ch *ParticipantHandler) GetChatParticipants(g *gin.Context) {
	cid := g.Query(dto.ChatIdQueryParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	participantsPage := utils.FixPageString(g.Query(dto.PageParam))
	participantsSize := utils.FixSizeString(g.Query(dto.SizeParam))
	participantsOffset := utils.GetOffset(participantsPage, participantsSize)

	ids, err := ch.commonProjection.GetParticipantIds(g.Request.Context(), ch.dbWrapper, chatId, participantsSize, participantsOffset)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error getting participant ids", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, ids)
	return
}

func (ch *ParticipantHandler) ParticipantsFilter(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	_, err = getUserId(g)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	bindTo := new(dto.FilteredParticipantsRequestDto)
	err = g.Bind(bindTo)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error during unmarshalling", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}
	requestedParticipantIds := bindTo.UserId
	userSearchString := bindTo.SearchString

	response, err := ch.enrichingProjection.ParticipantsFilter(g.Request.Context(), ch.dbWrapper, userSearchString, chatId, requestedParticipantIds)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error during ParticipantsFilter", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, response)
	return
}

func (ch *ParticipantHandler) SearchParticipants(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userId, err := getUserId(g)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	participantsPage := utils.FixPageString(g.Query(dto.PageParam))
	participantsSize := utils.FixSizeString(g.Query(dto.SizeParam))
	participantsOffset := utils.GetOffset(participantsPage, participantsSize)
	searchString := g.Query(dto.SearchStringParam)

	participantsByBehalfs, count, err := ch.enrichingProjection.GetParticipantsEnriched(g.Request.Context(), []int64{userId}, chatId, participantsSize, participantsOffset, searchString, true, nil)
	if err != nil {
		if translateParticipantError(g, err) {
			return
		}

		ch.lgr.ErrorContext(g.Request.Context(), "Error getting participants", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, dto.ParticipantsWithAdminWrapper{
		Data:  participantsByBehalfs[userId],
		Count: count,
	})
}

func (ch *ParticipantHandler) SearchForUsersToAdd(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userId, err := getUserId(g)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	searchString := g.Query(dto.SearchStringParam)

	users, err := ch.enrichingProjection.SearchUsersNotContainingForAdding(g.Request.Context(), ch.dbWrapper, userId, searchString, chatId, utils.DefaultSize)
	if err != nil {
		if translateParticipantError(g, err) {
			return
		}

		ch.lgr.ErrorContext(g.Request.Context(), "Error parsing searching", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, users)
	return
}

func (ch *ParticipantHandler) CountParticipants(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	_, err = getUserId(g)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	bindTo := new(dto.CountRequestDto)
	err = g.Bind(bindTo)
	if err != nil {
		ch.lgr.ErrorContext(g.Request.Context(), "Error binding CountRequestDto", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}
	userSearchString := bindTo.SearchString

	totalFoundUserCount := int64(0)

	if userSearchString != "" {
		_, aCount, err := ch.enrichingProjection.SearchUsersContaining(g.Request.Context(), ch.dbWrapper, userSearchString, chatId, utils.DefaultSize, utils.DefaultOffset, true, true)
		if err != nil {
			ch.lgr.ErrorContext(g.Request.Context(), "Error searchUsersContaining", logger.AttributeError, err)
			g.Status(http.StatusInternalServerError)
			return
		}
		totalFoundUserCount = aCount
	} else {
		count, err := ch.commonProjection.GetParticipantsCount(g.Request.Context(), ch.dbWrapper, chatId)
		if err != nil {
			ch.lgr.ErrorContext(g.Request.Context(), "Error GetParticipantsCount", logger.AttributeError, err)
			g.Status(http.StatusInternalServerError)
			return
		}
		totalFoundUserCount = count
	}

	g.JSON(http.StatusOK, &utils.H{"count": totalFoundUserCount})
	return
}

// returns should exit
func translateParticipantError(g *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var unauthError *cqrs.UnauthorizedError
	var chatStillNotExistsError *cqrs.ChatStillNotExistsError
	var participantsError *cqrs.ParticipantsError
	if errors.As(err, &unauthError) {
		g.JSON(http.StatusUnauthorized, &dto.ErrorMessageDto{unauthError.Error()})
		return true
	} else if errors.As(err, &chatStillNotExistsError) {
		g.Status(http.StatusTeapot)
		return true
	} else if errors.As(err, &participantsError) {
		g.Status(http.StatusBadRequest)
		return true
	}
	return false
}
