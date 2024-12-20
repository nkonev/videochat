package producer

import (
	"context"
	"encoding/json"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"nkonev.name/chat/logger"
	myRabbitmq "nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/utils"
	"time"
)

const EventsFanoutExchange = "async-events-exchange"
const NotificationsFanoutExchange = "notifications-exchange"

func (rp *RabbitEventsPublisher) Publish(ctx context.Context, aDto interface{}) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	aType := utils.GetType(aDto)

	bytea, err := json.Marshal(aDto)
	if err != nil {
		logger.GetLogEntry(ctx, rp.lgr).Error(err, "Failed during marshal dto")
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
		logger.GetLogEntry(ctx, rp.lgr).Error(err, "Error during publishing dto")
		return err
	} else {
		return nil
	}
}

type RabbitEventsPublisher struct {
	channel *rabbitmq.Channel
	lgr     *log.Logger
}

func NewRabbitEventsPublisher(lgr *log.Logger, connection *rabbitmq.Connection) *RabbitEventsPublisher {
	return &RabbitEventsPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
		lgr:     lgr,
	}
}

func (rp *RabbitNotificationsPublisher) Publish(ctx context.Context, aDto interface{}) error {
	headers := myRabbitmq.InjectAMQPHeaders(ctx)

	bytea, err := json.Marshal(aDto)
	if err != nil {
		logger.GetLogEntry(ctx, rp.lgr).Error(err, "Failed during marshal dto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now().UTC(),
		ContentType:  "application/json",
		Body:         bytea,
		Headers:      headers,
	}

	if err := rp.channel.Publish(NotificationsFanoutExchange, "", false, false, msg); err != nil {
		logger.GetLogEntry(ctx, rp.lgr).Error(err, "Error during publishing dto")
		return err
	} else {
		return nil
	}
}

type RabbitNotificationsPublisher struct {
	channel *rabbitmq.Channel
	lgr     *log.Logger
}

func NewRabbitNotificationsPublisher(lgr *log.Logger, connection *rabbitmq.Connection) *RabbitNotificationsPublisher {
	return &RabbitNotificationsPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(lgr, connection),
		lgr:     lgr,
	}
}
