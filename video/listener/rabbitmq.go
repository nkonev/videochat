package listener

import (
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/streadway/amqp"
	"time"
)

var logger = log.New()

const videoKickExchange = "video-kick"

func createFanoutExchange(name string, consume *rabbitmq.Channel) {
	var err error
	const maxRetries = 60
	var i = 0
	for ; i < maxRetries; i++ {
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
			logger.Info("Unable to declare to exchange, restarting.", "exchange", name, "error", err)
		} else {
			logger.Info("Successfully declared exchange", "exchange", name)
			break
		}
		duration, _ := time.ParseDuration("1s")
		time.Sleep(duration)
	}

	if i == maxRetries {
		logger.Error(err, "Unable to declare exchange after n retries", "exchange", name, "retries", maxRetries)
		panic(err)
	}
}

func bindQueueToExchange(exchangeName string, queue *amqp.Queue, consume *rabbitmq.Channel) {
	var err error
	const maxRetries = 60
	var i = 0
	for ; i < maxRetries; i++ {
		err = consume.QueueBind(
			queue.Name, // queue name
			"",     // routing key
			exchangeName, // exchange
			false,
			nil,
		)
		if err != nil {
			logger.Info("Unable to bind queue to exchange, restarting.", "exchange", exchangeName, "error", err)
		} else {
			logger.Info("Successfully bound queue to exchange", "exchange", exchangeName)
			break
		}
		duration, _ := time.ParseDuration("1s")
		time.Sleep(duration)
	}

	if i == maxRetries {
		logger.Error(err, "Unable to bind queue to exchange after n retries", "exchange", exchangeName, "retries", maxRetries)
		panic(err)
	}
}



func createAnonymousQueue(consumeCh *rabbitmq.Channel) *amqp.Queue {
	var err error
	var q amqp.Queue
	const maxRetries = 60
	var i = 0
	for ; i < maxRetries; i++ {
		q, err = consumeCh.QueueDeclare(
			"", // name
			false,   // durable
			false,   // delete when unused
			true,   // exclusive
			false,   // no-wait
			nil,     // arguments
		)
		if err != nil {
			logger.Info("Unable to declare to queue, restarting.",  "error", err)
		} else {
			logger.Info("Successfully declared queue", )
			break
		}
		duration, _ := time.ParseDuration("1s")
		time.Sleep(duration)
	}

	if i == maxRetries {
		logger.Error(err, "Unable to declare queue after n retries",  "retries", maxRetries)
		panic(err)
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
		const maxRetries = 60
		var i = 0
		for ; i < maxRetries; i++ {
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
				logger.Info("Unable to connect to queue, restarting", "queue", queue.Name, "error", err)
			} else {
				logger.Info("Successfully connected to queue", "queue", queue.Name)
				break
			}
			duration, _ := time.ParseDuration("1s")
			time.Sleep(duration)
		}
		if i == maxRetries {
			logger.Error(err, "Unable to connect to queue after n retries", "queue", queue.Name, "retries", maxRetries)
			panic(err)
		}

		for msg := range deliveries {
			onMessage(msg.Body)
		}
	}()
}

