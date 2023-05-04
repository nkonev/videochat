package producer

import (
	"context"
	"encoding/json"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"nkonev.name/storage/dto"
	. "nkonev.name/storage/logger"
	myRabbitmq "nkonev.name/storage/rabbitmq"
	"nkonev.name/storage/utils"
	"time"
)

const AsyncEventsFanoutExchange = "async-events-exchange"

func (rp *RabbitFileUploadedPublisher) Publish(userId, chatId int64, previewCreatedEvent *dto.PreviewCreatedEvent, ctx context.Context) error {

	event := dto.ChatEvent{
		EventType:           "preview_created",
		UserId:              userId,
		ChatId:              chatId,
		PreviewCreatedEvent: previewCreatedEvent,
	}

	bytea, err := json.Marshal(event)
	if err != nil {
		GetLogEntry(ctx).Error(err, "Failed during marshal PreviewCreatedEvent")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         utils.GetType(event),
	}

	if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
		GetLogEntry(ctx).Error(err, "Error during publishing")
		return err
	}

	return nil
}

func (rp *RabbitFileUploadedPublisher) PublishFileEvent(userId, chatId int64, fileInfoDto *dto.WrappedFileInfoDto, created bool, ctx context.Context) error {

	var eventType = "file_created"
	if !created {
		eventType = "file_removed"
	}
	event := dto.ChatEvent{
		EventType: eventType,
		UserId:    userId,
		ChatId:    chatId,
		FileEvent: fileInfoDto,
	}

	bytea, err := json.Marshal(event)
	if err != nil {
		GetLogEntry(ctx).Error(err, "Failed during marshal FileInfoDto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         utils.GetType(event),
	}

	if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
		GetLogEntry(ctx).Error(err, "Error during publishing")
		return err
	}

	return nil
}

type RabbitFileUploadedPublisher struct {
	channel *rabbitmq.Channel
}

func NewRabbitFileUploadedPublisher(connection *rabbitmq.Connection) *RabbitFileUploadedPublisher {
	return &RabbitFileUploadedPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(connection),
	}
}
