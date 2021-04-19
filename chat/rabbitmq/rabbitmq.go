package rabbitmq

import (
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	"github.com/spf13/viper"
	. "nkonev.name/chat/logger"
)

func CreateRabbitMqConnection() *rabbitmq.Connection{
	rabbitmq.Debug = true

	conn, err := rabbitmq.Dial(viper.GetString("rabbitmq.url"))
	if err != nil {
		Logger.Panic(err)
	}
	return conn
}

func CreateRabbitMqChannel(connection *rabbitmq.Connection) *rabbitmq.Channel{
	consumeCh, err := connection.Channel()
	if err != nil {
		Logger.Panic(err)
	}
	return consumeCh
}
