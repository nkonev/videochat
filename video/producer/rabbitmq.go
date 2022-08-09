package producer

import (
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	. "nkonev.name/video/logger"
	myRabbitmq "nkonev.name/video/rabbitmq"
	"time"
)

const videoNotificationsQueue = "video-notifications"
const videoInviteQueue = "video-invite"

func (rp *RabbitNotificationsPublisher) Publish(bytea []byte) error {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         bytea,
	}

	if err := rp.channel.Publish("", videoNotificationsQueue, false, false, msg); err != nil {
		Logger.Error(err, "Error during publishing")
		return err
	} else {
		return nil
	}
}

type RabbitNotificationsPublisher struct {
	channel *rabbitmq.Channel
}

func (rp *RabbitInvitePublisher) Publish(bytea []byte) error {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         bytea,
	}

	if err := rp.channel.Publish("", videoInviteQueue, false, false, msg); err != nil {
		Logger.Error(err, "Error during publishing")
		return err
	} else {
		return nil
	}
}

type RabbitInvitePublisher struct {
	channel *rabbitmq.Channel
}

func NewRabbitNotificationsPublisher(connection *rabbitmq.Connection) *RabbitNotificationsPublisher {
	return &RabbitNotificationsPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(connection),
	}
}

func NewRabbitInvitePublisher(connection *rabbitmq.Connection) *RabbitInvitePublisher {
	return &RabbitInvitePublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(connection),
	}
}
