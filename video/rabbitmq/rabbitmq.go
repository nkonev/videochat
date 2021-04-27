package rabbitmq

import (
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	log "github.com/pion/ion-sfu/pkg/logger"
	"nkonev.name/video/config"
)

type VideoListenerFunction func(data []byte) error

var logger = log.New()

func CreateRabbitMqConnection(conf config.RabbitMqConfig) *rabbitmq.Connection {
	rabbitmq.Debug = true

	conn, err := rabbitmq.Dial(conf.Url)
	if err != nil {
		logger.Error(err, "Unable to connect to rabbitmq")
		panic(err)
	}

	return conn
}

func CreateRabbitMqChannelWithRecreate(
	connection *rabbitmq.Connection,
	callback func(argChannel *rabbitmq.Channel) (error),
) *rabbitmq.Channel {
	channel, err := connection.Channel(callback)
	if err != nil {
		logger.Error(err, "Unable to create channel")
		panic(err)
	}
	return channel
}

func CreateRabbitMqChannel(
	connection *rabbitmq.Connection,
) *rabbitmq.Channel {
	channel, err := connection.Channel(func(argChannel *rabbitmq.Channel) error {
		return nil
	})
	if err != nil {
		logger.Error(err, "Unable to create channel")
		panic(err)
	}
	return channel
}


