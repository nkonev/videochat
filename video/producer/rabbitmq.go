package producer

import (
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/streadway/amqp"
	"nkonev.name/video/config"
	"time"
)

var logger = log.New()

const videoNotificationsQueue = "video-notifications"

func createRabbitMqConnection(url string) *rabbitmq.Channel{
	rabbitmq.Debug = true

	conn, err := rabbitmq.Dial(url)
	if err != nil {
		logger.Error(err, "Unable to connect to rabbitmq")
		panic(err)
	}

	channel, err := conn.Channel()
	if err != nil {
		logger.Error(err, "Unable to create channel")
		panic(err)
	}
	return channel
}

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

func NewRabbitPublisher(conf config.RabbitMqConfig) *RabbitPublisher {
	return &RabbitPublisher{
		channel: createRabbitMqConnection(conf.Url),
	}
}