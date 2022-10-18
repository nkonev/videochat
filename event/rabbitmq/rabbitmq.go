package rabbitmq

import (
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/spf13/viper"
	. "nkonev.name/event/logger"
)

func CreateRabbitMqConnection() *rabbitmq.Connection {
	rabbitmq.Debug = true

	conn, err := rabbitmq.Dial(viper.GetString("rabbitmq.url"))
	if err != nil {
		Logger.Panic(err)
	}
	return conn
}

func CreateRabbitMqChannel(connection *rabbitmq.Connection) *rabbitmq.Channel {
	consumeCh, err := connection.Channel(nil)
	if err != nil {
		Logger.Panic(err)
	}
	return consumeCh
}

func CreateRabbitMqChannelWithCallback(connection *rabbitmq.Connection, clbFunc rabbitmq.ChannelCallbackFunc) *rabbitmq.Channel {
	consumeCh, err := connection.Channel(clbFunc)
	if err != nil {
		Logger.Panic(err)
	}
	return consumeCh
}
