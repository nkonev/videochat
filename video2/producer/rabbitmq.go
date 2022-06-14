package producer

import (
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	. "nkonev.name/video/logger"
	myRabbitmq "nkonev.name/video/rabbitmq"
	"time"
)

const videoNotificationsQueue = "video-notifications"

func (rp *RabbitPublisher) Publish(bytea []byte) error {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
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

type RabbitPublisher struct {
	channel *rabbitmq.Channel
}

func NewRabbitPublisher(connection *rabbitmq.Connection) *RabbitPublisher {
	return &RabbitPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(connection),
	}
}
