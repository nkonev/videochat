package rabbitmq

import (
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
)

type VideoListenerFunction func(data []byte) error

func CreateRabbitMqConnection(conf *config.ExtendedConfig) *rabbitmq.Connection {
	rabbitmq.Debug = conf.RabbitMqConfig.Debug

	conn, err := rabbitmq.Dial(conf.RabbitMqConfig.Url)
	if err != nil {
		Logger.Error(err, "Unable to connect to rabbitmq")
		panic(err)
	}

	return conn
}

func CreateRabbitMqChannelWithRecreate(
	connection *rabbitmq.Connection,
	callback func(argChannel *rabbitmq.Channel) error,
) *rabbitmq.Channel {
	channel, err := connection.Channel(callback)
	if err != nil {
		Logger.Error(err, "Unable to create channel")
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
		Logger.Error(err, "Unable to create channel")
		panic(err)
	}
	return channel
}
