package rabbitmq

import (
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	log "github.com/pion/ion-sfu/pkg/logger"
	"nkonev.name/video/config"
)

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

func CreateRabbitMqChannel(connection *rabbitmq.Connection) *rabbitmq.Channel {
	channel, err := connection.Channel()
	if err != nil {
		logger.Error(err, "Unable to create channel")
		panic(err)
	}
	return channel
}

