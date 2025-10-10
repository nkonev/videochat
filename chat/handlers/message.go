package handlers

import (
	"errors"
	"net/http"
	"slices"

	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/sanitizer"
	"nkonev.name/chat/services"
	"nkonev.name/chat/utils"

	"github.com/gin-gonic/gin"
)

const badMediaUrl = "BAD_MEDIA_URL"

type MessageHandler struct {
	lgr                                 *logger.LoggerWrapper
	eventBus                            *cqrs.KafkaProducer
	dbWrapper                           *db.DB
	commonProjection                    *cqrs.CommonProjection
	policy                              *sanitizer.SanitizerPolicy
	stripAllTags                        *sanitizer.StripTagsPolicy
	cfg                                 *config.AppConfig
	enrichingProjection                 *cqrs.EnrichingProjection
	asyncMessageService                 *services.AsyncMessageService
	messageService                      *services.MessageService
	rabbitmqOutputEventPublisher        *producer.RabbitOutputEventsPublisher
	rabbitmqNotificationEventsPublisher *producer.RabbitNotificationEventsPublisher
}

func NewMessageHandler(
	lgr *logger.LoggerWrapper,
	eventBus *cqrs.KafkaProducer,
	dbWrapper *db.DB,
	commonProjection *cqrs.CommonProjection,
	policy *sanitizer.SanitizerPolicy,
	stripAllTags *sanitizer.StripTagsPolicy,
	cfg *config.AppConfig,
	enrichingProjection *cqrs.EnrichingProjection,
	asyncMessageService *services.AsyncMessageService, // we use async message service in order not to perform potentially heavyweight iterations in user-facing handles
	messageService *services.MessageService,
	rabbitmqOutputEventPublisher *producer.RabbitOutputEventsPublisher,
	rabbitmqNotificationEventsPublisher *producer.RabbitNotificationEventsPublisher,
) *MessageHandler {
	return &MessageHandler{
		lgr:                                 lgr,
		eventBus:                            eventBus,
		dbWrapper:                           dbWrapper,
		commonProjection:                    commonProjection,
		policy:                              policy,
		stripAllTags:                        stripAllTags,
		cfg:                                 cfg,
		enrichingProjection:                 enrichingProjection,
		asyncMessageService:                 asyncMessageService,
		messageService:                      messageService,
		rabbitmqOutputEventPublisher:        rabbitmqOutputEventPublisher,
		rabbitmqNotificationEventsPublisher: rabbitmqNotificationEventsPublisher,
	}
}

func (mc *MessageHandler) CreateMessage(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mcd := new(dto.MessageCreateDto)

	err = g.Bind(mcd)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding MessageCreateDto", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cc := cqrs.MessageCreate{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ChatId:         chatId,
		Content:        mcd.Content,
		FileItemUuid:   mcd.FileItemUuid,
	}
	if mcd.EmbedMessageRequest != nil {
		cc.EmbedMessage = &cqrs.EmbedMessage{
			Id:        mcd.EmbedMessageRequest.Id,
			ChatId:    mcd.EmbedMessageRequest.ChatId,
			EmbedType: mcd.EmbedMessageRequest.EmbedType,
		}
	}

	userPermissions := getUserPermissions(g)

	mid, err := cc.Handle(g.Request.Context(), mc.eventBus, mc.dbWrapper, mc.commonProjection, mc.cfg, mc.lgr, mc.policy, userPermissions)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MessageCreate command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	m := dto.IdResponse{Id: mid}

	g.JSON(http.StatusOK, m)
}

func (mc *MessageHandler) EditMessage(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)
	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	ccd := new(dto.MessageEditDto)

	err = g.Bind(ccd)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding MessageEditDto", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cc := cqrs.MessageEdit{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		MessageId:      ccd.Id,
		ChatId:         chatId,
		Content:        ccd.Content,
		FileItemUuid:   ccd.FileItemUuid,
	}
	if ccd.EmbedMessageRequest != nil {
		cc.EmbedMessage = &cqrs.EmbedMessage{
			Id:        ccd.EmbedMessageRequest.Id,
			ChatId:    ccd.EmbedMessageRequest.ChatId,
			EmbedType: ccd.EmbedMessageRequest.EmbedType,
		}
	}

	err = cc.Handle(g.Request.Context(), mc.eventBus, mc.dbWrapper, mc.commonProjection, mc.cfg, mc.lgr, mc.policy)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MessageEdit command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (mc *MessageHandler) SetFileItemUuid(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)
	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	ccd := new(dto.SetFileItemUuid)

	err = g.Bind(ccd)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding SetFileItemUuid", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cc := cqrs.MessageSetFileItemUuid{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		MessageId:      ccd.MessageId,
		ChatId:         chatId,
		FileItemUuid:   ccd.FileItemUuid,
	}

	err = cc.Handle(g.Request.Context(), mc.eventBus, mc.dbWrapper, mc.commonProjection, mc.cfg, mc.lgr, mc.policy)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending SetFileItemUuid command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (mc *MessageHandler) SyncEmbed(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)
	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mid := g.Param(dto.MessageIdParam)

	messageId, err := utils.ParseInt64(mid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding messageId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cc := cqrs.MessageSyncEmbed{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		MessageId:      messageId,
		ChatId:         chatId,
	}

	err = cc.Handle(g.Request.Context(), mc.eventBus, mc.dbWrapper, mc.commonProjection, mc.cfg, mc.lgr, mc.policy)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MessageEdit command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (mc *MessageHandler) DeleteMessage(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)
	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mid := g.Param(dto.MessageIdParam)
	messageId, err := utils.ParseInt64(mid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding messageId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cc := cqrs.MessageDelete{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		MessageId:      messageId,
		ChatId:         chatId,
	}

	err = cc.Handle(g.Request.Context(), mc.eventBus, mc.dbWrapper, mc.commonProjection)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MessageDelete command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (mc *MessageHandler) ReadMessage(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mid := g.Param(dto.MessageIdParam)

	messageId, err := utils.ParseInt64(mid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding messageId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mr := cqrs.MessageRead{
		AdditionalData:     cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ChatId:             chatId,
		MessageId:          messageId,
		ReadMessagesAction: cqrs.ReadMessagesActionOneMessage,
	}

	err = mr.Handle(g.Request.Context(), mc.lgr, mc.eventBus, mc.commonProjection, mc.dbWrapper, mc.rabbitmqOutputEventPublisher, mc.rabbitmqNotificationEventsPublisher)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MessageRead command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (mc *MessageHandler) MarkChatAsRead(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mr := cqrs.MessageRead{
		AdditionalData:     cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ReadMessagesAction: cqrs.ReadMessagesActionAllMessagesInOneChat,
		ChatId:             chatId,
	}

	err = mr.Handle(g.Request.Context(), mc.lgr, mc.eventBus, mc.commonProjection, mc.dbWrapper, mc.rabbitmqOutputEventPublisher, mc.rabbitmqNotificationEventsPublisher)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MessageRead command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (mc *MessageHandler) MarkAsReadAllChats(g *gin.Context) {

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mr := cqrs.MessageRead{
		AdditionalData:     cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ReadMessagesAction: cqrs.ReadMessagesActionAllChats,
	}

	err = mr.Handle(g.Request.Context(), mc.lgr, mc.eventBus, mc.commonProjection, mc.dbWrapper, mc.rabbitmqOutputEventPublisher, mc.rabbitmqNotificationEventsPublisher)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MessageRead command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (mc *MessageHandler) GetReadMessageUsers(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)
	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mid := g.Param(dto.MessageIdParam)

	messageId, err := utils.ParseInt64(mid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding messageId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	page := utils.FixPageString(g.Query(dto.PageParam))
	size := utils.FixSizeString(g.Query(dto.SizeParam))
	offset := utils.GetOffset(page, size)

	data, err := mc.enrichingProjection.GetReadMessageUsers(g.Request.Context(), userId, chatId, messageId, size, offset)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MessageRead command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, data)
}

func (mc *MessageHandler) ReactionMessage(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)
	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mid := g.Param(dto.MessageIdParam)

	messageId, err := utils.ParseInt64(mid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding messageId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	ccd := new(dto.ReactionPutDto)

	err = g.Bind(ccd)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding ReactionPutDto", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mr := cqrs.MessageReactionFlip{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ChatId:         chatId,
		MessageId:      messageId,
		Reaction:       ccd.Reaction,
	}

	err = mr.Handle(g.Request.Context(), mc.eventBus, mc.dbWrapper, mc.commonProjection, mc.policy)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MessageReactionFlip command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (mc *MessageHandler) TypeMessage(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userLogin, err := getUserLogin(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing userLogin", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)
	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	d := new(dto.BroadcastDto)

	err = g.Bind(d)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding BroadcastDto", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mc.asyncMessageService.TypeMessage(g.Request.Context(), chatId, userId, userLogin)

	g.Status(http.StatusOK)
	return
}

func (mc *MessageHandler) BroadcastMessage(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userLogin, err := getUserLogin(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing userLogin", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)
	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	d := new(dto.BroadcastDto)

	err = g.Bind(d)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding BroadcastDto", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	err = mc.asyncMessageService.BroadcastMessage(g.Request.Context(), d.Text, chatId, userId, userLogin)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error during broadcast message", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
	return
}

func (mc *MessageHandler) MessagesFresh(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	size := utils.FixSizeString(g.Query(dto.SizeParam))
	reverse := true // true for edge
	var startingFromItemId *int64 = nil
	includeStartingFrom := false
	searchString := g.Query(dto.SearchStringParam)

	var bindTo = make([]dto.MessageViewEnrichedDto, 0)
	if err := g.Bind(&bindTo); err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error during binding to dto", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	messageDtos, notAparticipant, _, err := mc.enrichingProjection.GetMessagesEnriched(g.Request.Context(), []int64{userId}, true, false, &userId, chatId, size, startingFromItemId, includeStartingFrom, reverse, searchString, nil, nil)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error getting messages", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	if notAparticipant {
		g.Status(http.StatusNoContent)
		return
	}

	edge := true

	aLen := min(len(messageDtos), len(bindTo))
	if len(bindTo) == 0 && len(messageDtos) != 0 {
		edge = false
	}

	for i := range aLen {
		currentMessage := messageDtos[i]
		gottenMessage := bindTo[i]
		if currentMessage.Id != gottenMessage.Id {
			edge = false
			break
		}

		// we strip tags because a (public) video link has "live" time parameter, which is changed between requests
		// it leads us to the false comparison
		// so we remove all the tags to mitigate this issue
		currentMsgText := mc.stripAllTags.Sanitize(currentMessage.Content)
		gottenMsgText := mc.stripAllTags.Sanitize(gottenMessage.Content)
		if currentMsgText != gottenMsgText {
			edge = false
			break
		}
		if !slices.EqualFunc(currentMessage.Reactions, gottenMessage.Reactions, func(reaction1 dto.Reaction, reaction2 dto.Reaction) bool {
			return reaction1.Reaction == reaction2.Reaction && reaction1.Count == reaction2.Count
		}) {
			edge = false
			break
		}
		if currentMessage.BlogPost != gottenMessage.BlogPost {
			edge = false
			break
		}
		if !utils.ComparePointers(currentMessage.UpdateDateTime, gottenMessage.UpdateDateTime) {
			edge = false
			break
		}
	}

	g.JSON(http.StatusOK, dto.FreshDto{
		Ok: edge,
	})
	return
}

func (mc *MessageHandler) MessagesFilter(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)
	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	d := new(dto.MessageFilterDto)
	err = g.Bind(d)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding MessageFilterDto", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	searchString := d.SearchString
	messageId := d.MessageId

	found, err := mc.enrichingProjection.MessageFilter(g.Request.Context(), mc.dbWrapper, userId, chatId, searchString, messageId)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error invoking MessageFilter", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, dto.FilterDto{
		Found: found,
	})
	return
}

func (mc *MessageHandler) MakeBlogPost(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userPermissions := getUserPermissions(g)

	cid := g.Param(dto.ChatIdParam)
	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mid := g.Param(dto.MessageIdParam)

	messageId, err := utils.ParseInt64(mid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding messageId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mr := cqrs.MakeMessageBlogPost{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ChatId:         chatId,
		MessageId:      messageId,
		BlogPost:       true,
	}

	err = mr.Handle(g.Request.Context(), mc.cfg, userPermissions, mc.eventBus, mc.dbWrapper, mc.commonProjection)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MakeMessageBlogPost command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (mc *MessageHandler) GetPinnedMessages(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	size := utils.FixSizeString(g.Query(dto.SizeParam))
	page := utils.FixPageString(g.Query(dto.PageParam))
	offset := utils.GetOffset(page, size)

	pm, cnt, err := mc.enrichingProjection.GetPinnedMessagesEnriched(g.Request.Context(), chatId, userId, offset, size)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error invoking GetPinnedMessages", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, dto.PinnedMessagesWrapper{
		Data:  pm,
		Count: cnt,
	})
}

func (mc *MessageHandler) GetPinnedPromotedMessage(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	pp, notAparticipant, err := mc.enrichingProjection.GetPinnedPromotedMessage(g.Request.Context(), chatId, userId)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error invoking GetPinnedPromotedMessage", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	if notAparticipant {
		g.Status(http.StatusNoContent)
		return
	}

	g.JSON(http.StatusOK, pp)
	return
}

func (mc *MessageHandler) PinMessage(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	p := g.Query(dto.PinParam)

	pin := utils.GetBoolean(p)

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mid := g.Param(dto.MessageIdParam)

	messageId, err := utils.ParseInt64(mid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding messageId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cc := cqrs.MessagePin{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ChatId:         chatId,
		MessageId:      messageId,
		Pin:            pin,
	}

	err = cc.Handle(g.Request.Context(), mc.eventBus, mc.dbWrapper, mc.commonProjection)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MessagePin command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (mc *MessageHandler) PublishMessage(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	p := g.Query(dto.PublishParam)

	publish := utils.GetBoolean(p)

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mid := g.Param(dto.MessageIdParam)

	messageId, err := utils.ParseInt64(mid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding messageId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cc := cqrs.MessagePublish{
		AdditionalData: cqrs.GenerateMessageAdditionalData(getCorrelationId(g), userId),
		ChatId:         chatId,
		MessageId:      messageId,
		Publish:        publish,
	}

	err = cc.Handle(g.Request.Context(), mc.eventBus, mc.dbWrapper, mc.commonProjection)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error sending MessagePublish command", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
}

func (mc *MessageHandler) GetPublishedMessages(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	size := utils.FixSizeString(g.Query(dto.SizeParam))
	page := utils.FixPageString(g.Query(dto.PageParam))
	offset := utils.GetOffset(page, size)

	pm, cnt, err := mc.enrichingProjection.GetPublishedMessagesEnriched(g.Request.Context(), chatId, userId, offset, size)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error invoking GetPublishedMessages", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, dto.PublishedMessagesWrapper{
		Data:  pm,
		Count: cnt,
	})
}

func (mc *MessageHandler) GetPublishedMessageForPublic(g *gin.Context) {
	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	mid := g.Param(dto.MessageIdParam)
	messageId, err := utils.ParseInt64(mid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding messageId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	msg, notFound, err := mc.enrichingProjection.GetPublishedMessageForPublic(g.Request.Context(), chatId, messageId)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error invoking GetPublishedMessage", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	if notFound {
		g.Status(http.StatusNoContent)
		return
	}

	g.JSON(http.StatusOK, msg)
}

func (mc *MessageHandler) SearchMessages(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	size := utils.FixSizeString(g.Query(dto.SizeParam))
	reverse := utils.GetBoolean(g.Query(dto.ReverseParam))
	startingFromItemIdString := g.Query(dto.StartingFromItemId)
	var startingFromItemId *int64
	if startingFromItemIdString != "" {
		startingFromItemId2, err := utils.ParseInt64(startingFromItemIdString) // exclusive
		if err != nil {
			mc.lgr.ErrorContext(g.Request.Context(), "Error parsing startingFromItemId", logger.AttributeError, err)
			g.Status(http.StatusInternalServerError)
			return
		}
		startingFromItemId = &startingFromItemId2
	}
	includeStartingFrom := utils.GetBoolean(g.Query(dto.IncludeStartingFromParam))
	searchString := g.Query(dto.SearchStringParam)

	messages, notAparticipant, _, err := mc.enrichingProjection.GetMessagesEnriched(g.Request.Context(), []int64{userId}, true, false, &userId, chatId, size, startingFromItemId, includeStartingFrom, reverse, searchString, nil, nil)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error getting messages", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	if notAparticipant {
		g.Status(http.StatusNoContent)
		return
	}

	g.JSON(http.StatusOK, dto.MessagesResponseDto{
		Items:   messages,
		HasNext: int32(len(messages)) == size,
	})
}

func (mc *MessageHandler) MessagePreview(g *gin.Context) {
	bindTo := new(dto.CleanHtmlTagsRequestDto)
	err := g.Bind(bindTo)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding CleanHtmlTagsRequestDto", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	preview := mc.messageService.CreatePreview(bindTo.Text, bindTo.Login)
	response := dto.CleanHtmlTagsResponseDto{
		Text: preview,
	}
	g.JSON(http.StatusOK, response)
}

func (mc *MessageHandler) SearchForUsersToMention(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	searchString := g.Query(dto.SearchStringParam)

	res, err := mc.messageService.SearchForUsersToMention(g.Request.Context(), chatId, userId, searchString)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error getting messages", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, res)
}

func (mc *MessageHandler) FindMessageByFileItemUuid(g *gin.Context) {
	userId, err := getUserId(g)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error parsing UserId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	cid := g.Param(dto.ChatIdParam)

	chatId, err := utils.ParseInt64(cid)
	if err != nil {
		mc.lgr.ErrorContext(g.Request.Context(), "Error binding chatId", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	fileItemUuid := g.Param("fileItemUuid")

	res, err := mc.commonProjection.FindMessageByFileItemUuid(g.Request.Context(), chatId, userId, fileItemUuid)
	if err != nil {
		if translateMessageError(g, err) {
			return
		}

		mc.lgr.ErrorContext(g.Request.Context(), "Error getting messages", logger.AttributeError, err)
		g.Status(http.StatusInternalServerError)
		return
	}

	g.JSON(http.StatusOK, res)
}

// returns should exit
func translateMessageError(g *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var mediaError *sanitizer.MediaUrlErr
	var mediaOverflowError *sanitizer.MediaOverflowErr
	var validationError *cqrs.ValidationError
	var chatStillNotExistsError *cqrs.ChatStillNotExistsError
	var messageStillNotExistsError *cqrs.MessageStillNotExistsError
	var unauthError *cqrs.UnauthorizedError
	if errors.As(err, &mediaError) {
		g.JSON(http.StatusBadRequest, &utils.H{"message": mediaError.Error(), "businessErrorCode": badMediaUrl})
		return true
	} else if errors.As(err, &mediaOverflowError) {
		g.JSON(http.StatusBadRequest, &dto.ErrorMessageDto{mediaOverflowError.Error()})
		return true
	} else if errors.As(err, &validationError) {
		g.JSON(http.StatusBadRequest, &dto.ErrorMessageDto{validationError.Error()})
		return true
	} else if errors.As(err, &chatStillNotExistsError) {
		g.Status(http.StatusTeapot)
		return true
	} else if errors.As(err, &messageStillNotExistsError) {
		g.Status(http.StatusTeapot)
		return true
	} else if errors.As(err, &unauthError) {
		g.JSON(http.StatusUnauthorized, &dto.ErrorMessageDto{unauthError.Error()})
		return true
	}
	return false
}
