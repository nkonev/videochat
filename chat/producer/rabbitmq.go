package producer

import (
	"context"
	"encoding/json"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"nkonev.name/chat/logger"
	myRabbitmq "nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/utils"
	"time"
)

const EventsFanoutExchange = "async-events-exchange"
const NotificationsPersistentFanoutExchange = "notifications-persistent-exchange"

func (rp *RabbitEventsPublisher) Publish(ctx context.Context, aDto interface{}) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	aType := utils.GetType(aDto)

	bytea, err := json.Marshal(aDto)
	if err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal dto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         aType,
		Headers:      headers,
	}

	if err := rp.channel.Publish(EventsFanoutExchange, "", false, false, msg); err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Error during publishing dto")
		return err
	} else {
		return nil
	}
}

type RabbitEventsPublisher struct {
	channel *rabbitmq.Channel
	lgr     *logger.Logger
}

func NewRabbitEventsPublisher(lgr *logger.Logger, connection *rabbitmq.Connection) *RabbitEventsPublisher {
	return &RabbitEventsPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
		lgr:     lgr,
	}
}

func (rp *RabbitNotificationsPublisher) Publish(ctx context.Context, aDto interface{}) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	bytea, err := json.Marshal(aDto)
	if err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Failed during marshal dto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Headers:      headers,
	}

	if err := rp.channel.Publish(NotificationsPersistentFanoutExchange, "", false, false, msg); err != nil {
		rp.lgr.WithTracing(ctx).Error(err, "Error during publishing dto")
		return err
	} else {
		return nil
	}
}

type RabbitNotificationsPublisher struct {
	channel *rabbitmq.Channel
	lgr     *logger.Logger
}

func NewRabbitNotificationsPublisher(lgr *logger.Logger, connection *rabbitmq.Connection) *RabbitNotificationsPublisher {
	return &RabbitNotificationsPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
		lgr:     lgr,
	}
}
