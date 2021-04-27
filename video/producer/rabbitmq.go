package producer

import (
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/streadway/amqp"
	"time"
	myRabbitmq "nkonev.name/video/rabbitmq"
)

var logger = log.New()

const videoNotificationsQueue = "video-notifications"

func (rp *RabbitPublisher) Publish(bytea []byte) error {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body: bytea,
	}

	if err := rp.channel.Publish("", videoNotificationsQueue, false, false, msg); err != nil {
		logger.Error(err, "Error during publishing")
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