package producer

import (
	"context"
	"encoding/json"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"nkonev.name/notification/dto"
	. "nkonev.name/notification/logger"
	myRabbitmq "nkonev.name/notification/rabbitmq"
	"nkonev.name/notification/utils"
	"time"
)

const AsyncEventsFanoutExchange = "async-events-exchange"

func (rp *RabbitEventPublisher) Publish(participantId int64, notifyDto *dto.NotificationDto, eventType string, ctx context.Context) error {

	event := dto.UserEvent{
		EventType:             eventType,
		UserId:                participantId,
		UserNotificationEvent: notifyDto,
	}

	bytea, err := json.Marshal(event)
	if err != nil {
		GetLogEntry(ctx).Error(err, "Failed during marshal NotificationDto")
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

type RabbitEventPublisher struct {
	channel *rabbitmq.Channel
}

func NewRabbiEventPublisher(connection *rabbitmq.Connection) *RabbitEventPublisher {
	return &RabbitEventPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(connection),
	}
}
