package producer

import (
	"context"
	"encoding/json"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"nkonev.name/notification/dto"
	"nkonev.name/notification/logger"
	myRabbitmq "nkonev.name/notification/rabbitmq"
	"nkonev.name/notification/utils"
	"time"
)

const AsyncEventsFanoutExchange = "async-events-exchange"

func (rp *RabbitEventPublisher) Publish(ctx context.Context, participantId int64, notifyDto *dto.WrapperNotificationDto, eventType string) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	event := dto.GlobalUserEvent{
		EventType:             eventType,
		UserId:                participantId,
		UserNotificationEvent: notifyDto,
	}

	bytea, err := json.Marshal(event)
	if err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal NotificationDto")
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

type RabbitEventPublisher struct {
	channel *rabbitmq.Channel
	lgr     *logger.Logger
}

func NewRabbiEventPublisher(lgr *logger.Logger, connection *rabbitmq.Connection) *RabbitEventPublisher {
	return &RabbitEventPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
	}
}
