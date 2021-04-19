package listener

import (
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/streadway/amqp"
)

var logger = log.New()

const videoKickExchange = "video-kick"

func createFanoutExchange(name string, consume *rabbitmq.Channel) {
	var err error
	err = consume.ExchangeDeclare(
		name,   // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		logger.Error(err, "Unable to declare to exchange, restarting.", "exchange", name)
		panic(err)
	} else {
		logger.Info("Successfully declared exchange", "exchange", name)
	}
}

func bindQueueToExchange(exchangeName string, queue *amqp.Queue, consume *rabbitmq.Channel) {
	var err error
	err = consume.QueueBind(
		queue.Name, // queue name
		"",     // routing key
		exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		logger.Error(err, "Unable to bind queue to exchange, restarting.", "exchange", exchangeName)
		panic(err)
	} else {
		logger.Info("Successfully bound queue to exchange", "exchange", exchangeName)
	}
}


func createAnonymousQueue(consumeCh *rabbitmq.Channel) *amqp.Queue {
	var err error
	var q amqp.Queue
	q, err = consumeCh.QueueDeclare(
		"", // name
		false,   // durable
		false,   // delete when unused
		true,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		logger.Error(err, "Unable to declare to queue, restarting.")
		panic(err)
	} else {
		logger.Info("Successfully declared queue", )
	}
	return &q
}

func listenQueue(
	channel *rabbitmq.Channel,
	queue *amqp.Queue,
	onMessage func(data []byte) error,
	) {
	logger.Info("Listening queue", "queue", queue.Name)
	go func() {
		var deliveries <-chan amqp.Delivery
		var err error
		deliveries, err = channel.Consume(
			queue.Name,
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			logger.Error(err, "Unable to connect to queue, restarting", "queue", queue.Name)
			panic(err)
		} else {
			logger.Info("Successfully connected to queue", "queue", queue.Name)
		}

		for msg := range deliveries {
			onMessage(msg.Body)
		}
	}()
}

