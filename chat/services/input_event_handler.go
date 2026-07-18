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
	"nkonev.name/chat/producer"
	"nkonev.name/chat/sanitizer"
)

type InputEventHandler struct {
	commonProjection             *cqrs.CommonProjection
	dba                          *db.DB
	lgr                          *logger.LoggerWrapper
	tr                           trace.Tracer
	rabbitmqOutputEventPublisher *producer.RabbitOutputEventsPublisher
	cfg                          *config.AppConfig
	eventBus                     *cqrs.KafkaProducer
	stripTagsPolicy              *sanitizer.StripTagsPolicy
}

func NewInputEventHandler(
	commonProjection *cqrs.CommonProjection,
	dba *db.DB,
	lgr *logger.LoggerWrapper,
	rabbitmqEventPublisher *producer.RabbitOutputEventsPublisher,
	cfg *config.AppConfig,
	eventBus *cqrs.KafkaProducer,
	stripTagsPolicy *sanitizer.StripTagsPolicy,
) *InputEventHandler {
	tr := otel.Tracer("event")

	return &InputEventHandler{
		commonProjection:             commonProjection,
		dba:                          dba,
		lgr:                          lgr,
		tr:                           tr,
		rabbitmqOutputEventPublisher: rabbitmqEventPublisher,
		cfg:                          cfg,
		eventBus:                     eventBus,
		stripTagsPolicy:              stripTagsPolicy,
	}
}

func (not InputEventHandler) CreateTetATetSelfIfNeed(ctx context.Context, userId int64) {
	if !not.cfg.Chat.TetATet.AutoCreateSelf {
		return
	}

	exists, _, err := not.commonProjection.IsExistsTetATetOne(ctx, not.dba, userId)
	if err != nil {
		not.lgr.ErrorContext(ctx, "Error during IsExistsTetATetOne", logger.AttributeError, err)
		return
	}

	if !exists {
		cc := cqrs.NewTetATetChatCreate(userId, userId, cqrs.GenerateMessageAdditionalData(nil, userId), not.cfg.Chat.TetATet)

		_, err = cc.Handle(ctx, not.eventBus, not.dba, not.commonProjection, not.stripTagsPolicy, not.cfg, not.rabbitmqOutputEventPublisher, not.lgr)
		if err != nil {
			not.lgr.ErrorContext(ctx, "Error during handling command", logger.AttributeError, err)
			return
		}

		not.lgr.InfoContext(ctx, "Created self tet-a-tet chat", logger.AttributeUserId, userId)
	}
}

func (not InputEventHandler) NotifyAboutProfileChanged(ctx context.Context, user *dto.User) {
	if user == nil {
		not.lgr.ErrorContext(ctx, "user cannot be null")
		return
	}

	eventType := dto.EventTypeParticipantChanged
	ctx, messageSpan := not.tr.Start(ctx, fmt.Sprintf("global.user.%s", eventType))
	defer messageSpan.End()

	err := not.commonProjection.IterateOverCoChattedParticipantIds(ctx, not.dba, user.Id, func(participantIds []int64) error {
		var internalErr error
		for _, participantId := range participantIds {
			internalErr = not.rabbitmqOutputEventPublisher.Publish(ctx, nil, dto.GlobalUserEvent{
				UserId:                           participantId,
				EventType:                        eventType,
				CoChattedParticipantNotification: user,
			})
		}
		return internalErr
	})
	if err != nil {
		not.lgr.ErrorContext(ctx, "Error during sending to rabbitmq", logger.AttributeError, err)
	}
}
