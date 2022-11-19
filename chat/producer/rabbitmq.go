package producer

import (
	"encoding/json"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	. "nkonev.name/chat/logger"
	myRabbitmq "nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/utils"
	"time"
)

const EventsFanoutExchange = "async-events-exchange"
const NotificationsFanoutExchange = "notifications-exchange"

func (rp *RabbitEventsPublisher) Publish(aDto interface{}) error {
	aType := utils.GetType(aDto)

	bytea, err := json.Marshal(aDto)
	if err != nil {
		Logger.Error(err, "Failed during marshal dto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         aType,
	}

	if err := rp.channel.Publish(EventsFanoutExchange, "", false, false, msg); err != nil {
		Logger.Error(err, "Error during publishing dto")
		return err
	} else {
		return nil
	}
}

type RabbitEventsPublisher struct {
	channel *rabbitmq.Channel
}

func NewRabbitEventsPublisher(connection *rabbitmq.Connection) *RabbitEventsPublisher {
	return &RabbitEventsPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(connection),
	}
}

func (rp *RabbitNotificationsPublisher) Publish(aDto interface{}) error {
	aType := utils.GetType(aDto)

	bytea, err := json.Marshal(aDto)
	if err != nil {
		Logger.Error(err, "Failed during marshal dto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         aType,
	}

	if err := rp.channel.Publish(NotificationsFanoutExchange, "", false, false, msg); err != nil {
		Logger.Error(err, "Error during publishing dto")
		return err
	} else {
		return nil
	}
}

type RabbitNotificationsPublisher struct {
	channel *rabbitmq.Channel
}

func NewRabbitNotificationsPublisher(connection *rabbitmq.Connection) *RabbitNotificationsPublisher {
	return &RabbitNotificationsPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(connection),
	}
}
