package services

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/chat/config"
	"nkonev.name/chat/cqrs"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/preview"
	"nkonev.name/chat/producer"
	"nkonev.name/chat/sanitizer"
	"nkonev.name/chat/utils"
)

type AsyncMessageService struct {
	lgr                          *logger.LoggerWrapper
	tr                           trace.Tracer
	rabbitmqOutputEventPublisher *producer.RabbitInternalEventsPublisher
	dbWrapper                    *db.DB
	commonProjection             *cqrs.CommonProjection
}

func NewAsyncMessageService(
	lgr *logger.LoggerWrapper,
	rabbitmqEventPublisher *producer.RabbitInternalEventsPublisher,
	dbWrapper *db.DB,
	commonProjection *cqrs.CommonProjection,
) *AsyncMessageService {
	tr := otel.Tracer("event")

	return &AsyncMessageService{
		lgr:                          lgr,
		tr:                           tr,
		rabbitmqOutputEventPublisher: rabbitmqEventPublisher,
		commonProjection:             commonProjection,
		dbWrapper:                    dbWrapper,
	}
}
func (p *AsyncMessageService) BroadcastMessage(ctx context.Context, messageText string, chatId, userId int64, userLogin string) error {
	adt, err := p.commonProjection.GetChatDataForAuthorization(ctx, p.dbWrapper, userId, chatId)
	if err != nil {
		return err
	}

	if !cqrs.CanBroadcast(adt.IsChatAdmin) {
		return cqrs.NewUnauthorizedError(fmt.Sprintf("user %v is not authorized to broadcast in the chat %v", userId, chatId))
	}

	err = p.rabbitmqOutputEventPublisher.Publish(ctx, dto.PublishBroadcastMessage{
		MessageText: messageText,
		ChatId:      chatId,
		UserId:      userId,
		UserLogin:   userLogin,
	})
	if err != nil {
		p.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
	}
	return nil
}

func (p *AsyncMessageService) TypeMessage(ctx context.Context, chatId, userId int64, userLogin string) {
	participant, err := p.commonProjection.IsParticipant(ctx, p.dbWrapper, userId, chatId)
	if err != nil {
		p.lgr.ErrorContext(ctx, "Error checking is participant", logger.AttributeError, err)
		return
	}
	if !participant {
		p.lgr.InfoContext(ctx, fmt.Sprintf("User %v is not participant of chat %v, skipping", userId, chatId))
		return
	}

	err = p.rabbitmqOutputEventPublisher.Publish(ctx, dto.PublishUserTyping{
		ChatId:    chatId,
		UserId:    userId,
		UserLogin: userLogin,
	})
	if err != nil {
		p.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
	}
}

type MessageService struct {
	lgr                          *logger.LoggerWrapper
	dbWrapper                    *db.DB
	commonProjection             *cqrs.CommonProjection
	stripAllTags                 *sanitizer.StripTagsPolicy
	policy                       *sanitizer.SanitizerPolicy
	cfg                          *config.AppConfig
	tr                           trace.Tracer
	rabbitmqOutputEventPublisher *producer.RabbitOutputEventsPublisher
	enrichingProjection          *cqrs.EnrichingProjection
}

func NewMessageService(
	lgr *logger.LoggerWrapper,
	dbWrapper *db.DB,
	commonProjection *cqrs.CommonProjection,
	stripAllTags *sanitizer.StripTagsPolicy,
	policy *sanitizer.SanitizerPolicy,
	cfg *config.AppConfig,
	rabbitmqEventPublisher *producer.RabbitOutputEventsPublisher,
	enrichingProjection *cqrs.EnrichingProjection,
) *MessageService {
	tr := otel.Tracer("event")

	return &MessageService{
		lgr:                          lgr,
		dbWrapper:                    dbWrapper,
		commonProjection:             commonProjection,
		stripAllTags:                 stripAllTags,
		policy:                       policy,
		cfg:                          cfg,
		tr:                           tr,
		rabbitmqOutputEventPublisher: rabbitmqEventPublisher,
		enrichingProjection:          enrichingProjection,
	}
}

func (p *MessageService) BroadcastMessage(ctx context.Context, messageText string, chatId, userId int64, userLogin string) {
	previewStr := preview.CreateMessagePreview(p.stripAllTags, p.cfg.Message.PreviewMaxTextSize, messageText, userLogin)
	if previewStr == preview.LoginPrefix(userLogin) {
		previewStr = ""
	}

	eventType := dto.EventTypeMessageBroadCast
	ctx, messageSpan := p.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	ut := dto.MessageBroadcastNotification{
		Login:  userLogin,
		UserId: userId,
		Text:   previewStr,
	}

	err := p.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, p.dbWrapper, chatId, []int64{}, func(participantIds []int64) error {
		for _, participantId := range participantIds {
			err := p.rabbitmqOutputEventPublisher.Publish(ctx, nil, dto.ChatEvent{
				EventType:                    eventType,
				MessageBroadcastNotification: &ut,
				UserId:                       participantId,
				ChatId:                       chatId,
			})
			if err != nil {
				p.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
			}
		}
		return nil
	})
	if err != nil {
		p.lgr.ErrorContext(ctx, "Error during getting chat participants", logger.AttributeError, err)
		return
	}
}

func (p *MessageService) TypeMessage(ctx context.Context, chatId, userId int64, userLogin string) {
	eventType := dto.EventTypeMessageType
	ctx, messageSpan := p.tr.Start(ctx, fmt.Sprintf("chat.%s", eventType))
	defer messageSpan.End()

	ut := dto.UserTypingNotification{
		Login:         userLogin,
		ParticipantId: userId,
		ChatId:        chatId,
	}

	participant, err := p.commonProjection.IsParticipant(ctx, p.dbWrapper, userId, chatId)
	if err != nil {
		p.lgr.ErrorContext(ctx, "Error during checking is participant", logger.AttributeError, err)
		return
	}

	if !participant {
		p.lgr.InfoContext(ctx, "The user isn't participant", logger.AttributeUserId, userId, logger.AttributeChatId, chatId)
		return
	}

	err = p.commonProjection.IterateOverChatParticipantIdsExcepting(ctx, p.dbWrapper, chatId, []int64{userId}, func(participantIds []int64) error {
		for _, participantId := range participantIds {
			err := p.rabbitmqOutputEventPublisher.Publish(ctx, nil, dto.GlobalUserEvent{
				UserId:                 participantId,
				EventType:              eventType,
				UserTypingNotification: &ut,
			})
			if err != nil {
				p.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
			}
		}
		return nil
	})
	if err != nil {
		p.lgr.ErrorContext(ctx, "Error during getting chat participants", logger.AttributeError, err)
		return
	}
}

func (p *MessageService) CreatePreview(messageText, userLogin string) string {

	var input string
	if len(userLogin) > 0 {
		input = preview.LoginPrefix(userLogin) + messageText
	} else {
		input = messageText
	}

	return preview.CreateMessagePreviewWithoutLogin(p.stripAllTags, p.cfg.Message.PreviewMaxTextSize, input)
}

func (p *MessageService) SearchForUsersToMention(ctx context.Context, chatId, userId int64, searchString string) ([]*dto.User, error) {
	participant, err := p.commonProjection.IsParticipant(ctx, p.dbWrapper, userId, chatId)
	if err != nil {
		return nil, err
	}
	if !participant {
		return nil, cqrs.NewUnauthorizedError(fmt.Sprintf("user %v is not a participant of chat %v", userId, chatId))
	}

	searchString = sanitizer.TrimAmdSanitize(p.policy, searchString)

	usersWithAdmin, _, err := p.enrichingProjection.SearchUsersContaining(ctx, p.dbWrapper, searchString, chatId, utils.DefaultSize, utils.DefaultOffset, true, false)
	if err != nil {
		return nil, err
	}

	users := []*dto.User{}
	for _, u := range usersWithAdmin {
		uu := u.User
		users = append(users, &uu)
	}

	users = append(users, &dto.User{
		Id:    dto.AllUsers, // -1 is reserved for 'deleted' in ./aaa/src/main/resources/db/migration/V1__init.sql
		Login: dto.AllUsersLogin,
	})
	users = append(users, &dto.User{
		Id:    dto.HereUsers, // -1 is reserved for 'deleted' in ./aaa/src/main/resources/db/migration/V1__init.sql
		Login: dto.HereUsersLogin,
	})

	return users, nil
}
