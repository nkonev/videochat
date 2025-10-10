package producer

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
	myRabbitmq "nkonev.name/storage/rabbitmq"
	"nkonev.name/storage/utils"
	"time"
)

const AsyncEventsFanoutExchange = "async-events-exchange"
const correlationIdName = "correlationId"

func (rp *RabbitFileUploadedPublisher) Publish(ctx context.Context, userId, chatId int64, previewCreatedEvent *dto.PreviewCreatedEvent, correlationId *string) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	if correlationId != nil && *correlationId != "" {
		headers[correlationIdName] = *correlationId
	}

	event := dto.ChatEvent{
		EventType:           "preview_created",
		UserId:              userId,
		ChatId:              chatId,
		PreviewCreatedEvent: previewCreatedEvent,
	}

	bytea, err := json.Marshal(event)
	if err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal PreviewCreatedEvent")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         utils.GetType(event),
		Headers:      headers,
	}

	if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Error during publishing")
		return err
	}

	return nil
}

func (rp *RabbitFileUploadedPublisher) PublishFileEvent(ctx context.Context, userId, chatId int64, fileInfoDto *dto.WrappedFileInfoDto, eventType utils.EventType, correlationId *string) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)
	if correlationId != nil && *correlationId != "" {
		headers[correlationIdName] = *correlationId
	}

	var outputEventType string
	switch eventType {
	case utils.FILE_CREATED:
		outputEventType = "file_created"
	case utils.FILE_DELETED:
		outputEventType = "file_removed"
	case utils.FILE_UPDATED:
		outputEventType = "file_updated"
	default:
		rp.lgr.WithTracing(ctx).Errorf("Error during determining rabbitmq output event type")
		return errors.New("Unknown type")
	}

	event := dto.ChatEvent{
		EventType: outputEventType,
		UserId:    userId,
		ChatId:    chatId,
		FileEvent: fileInfoDto,
	}

	bytea, err := json.Marshal(event)
	if err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal FileInfoDto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         utils.GetType(event),
		Headers:      headers,
	}

	if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Error during publishing")
		return err
	}

	return nil
}

type RabbitFileUploadedPublisher struct {
	channel *rabbitmq.Channel
	lgr     *logger.Logger
}

func NewRabbitFileUploadedPublisher(connection *rabbitmq.Connection, lgr *logger.Logger) *RabbitFileUploadedPublisher {
	return &RabbitFileUploadedPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
	}
}
