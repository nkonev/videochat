package listener

import (
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"go.uber.org/fx"
	. "nkonev.name/chat/logger"
)

const aaaEventsQueue = "aaa-events"

func failOnError(err error, msg string) {
	if err != nil {
		Logger.Fatalf("%s: %s", msg, err)
	}
}

func CreateRabbitMqConnection() (*rabbitmq.Channel, *amqp.Queue){
	rabbitmq.Debug = true

	conn, err := rabbitmq.Dial(viper.GetString("rabbitmq.url"))
	if err != nil {
		Logger.Panic(err)
	}

	consumeCh, err := conn.Channel()
	if err != nil {
		Logger.Panic(err)
	}

	q, err := consumeCh.QueueDeclare(
		aaaEventsQueue, // name
		true,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")
	return consumeCh, &q
}

func ListenPubSubChannels(
	channel *rabbitmq.Channel,
	queue *amqp.Queue,
	onMessage AaaUserProfileUpdateListener,
	lc fx.Lifecycle) {
	forever := make(chan bool)

	go func() {
		deliveries, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
		if err != nil {
			Logger.Panic(err)
		}

		for msg := range deliveries {
			onMessage(msg.Body)
			msg.Ack(true)
		}
	}()

	<-forever
}