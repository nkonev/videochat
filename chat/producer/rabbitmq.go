package producer

import (
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"time"
	myRabbitmq "nkonev.name/chat/rabbitmq"
	. "nkonev.name/chat/logger"
)

const videoKickExchange = "video-kick"

func (rp *RabbitPublisher) Publish(bytea []byte) error {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body: bytea,
	}

	if err := rp.channel.Publish(videoKickExchange, "", false, false, msg); err != nil {
		Logger.Error(err, "Error during publishing")
		return err
	} else {
		return nil
	}
}

type RabbitPublisher struct {
	channel *rabbitmq.Channel
}

type VideoKickChannel struct {*rabbitmq.Channel}

func CreateVideoKickChannel(connection *rabbitmq.Connection) VideoKickChannel {
	return VideoKickChannel{myRabbitmq.CreateRabbitMqChannel(connection)}
}

func NewRabbitPublisher(channel VideoKickChannel) *RabbitPublisher {
	return &RabbitPublisher{
		channel: channel.Channel,
	}
}