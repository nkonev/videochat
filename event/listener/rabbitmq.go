package listener

import (
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"go.uber.org/fx"
	. "nkonev.name/event/logger"
	myRabbit "nkonev.name/event/rabbitmq"
)

const AsyncEventsFanoutExchange = "async-events-exchange"

type FanoutNotificationsChannel struct{ *rabbitmq.Channel }

func create(name string, consumeCh *rabbitmq.Channel) *amqp.Queue {
	var err error
	var q amqp.Queue
	q, err = consumeCh.QueueDeclare(
		name,  // name
		true,  // durable - it prevents queue loss on rabbitmq restart
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		Logger.Warnf("Unable to declare to queue %v, restarting. error %v", name, err)
		Logger.Panic(err)
	}
	return &q
}

func createAndBind(name string, key string, exchange string, consumeCh *rabbitmq.Channel) *amqp.Queue {
	var err error
	var q amqp.Queue
	q, err = consumeCh.QueueDeclare(
		name,  // name
		true,  // durable - it prevents queue loss on rabbitmq restart
		true,  // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		Logger.Warnf("Unable to declare to queue %v, restarting. error %v", name, err)
		Logger.Panic(err)
	}
	err = consumeCh.QueueBind(q.Name, key, exchange, false, nil)
	if err != nil {
		Logger.Warnf("Unable to bind to queue %v, restarting. error %v", name, err)
		Logger.Panic(err)
	}
	return &q
}

func CreateEventsChannel(connection *rabbitmq.Connection, onMessage EventsListener, lc fx.Lifecycle) FanoutNotificationsChannel {
	var fanoutQueueName = "async-events-" + uuid.New().String()

	return FanoutNotificationsChannel{myRabbit.CreateRabbitMqChannelWithCallback(
		connection,
		func(channel *rabbitmq.Channel) error {
			err := channel.ExchangeDeclare(AsyncEventsFanoutExchange, "fanout", true, false, false, false, nil)
			if err != nil {
				return err
			}

			tempQueue := createAndBind(fanoutQueueName, "", AsyncEventsFanoutExchange, channel)
			listen(channel, tempQueue, onMessage, lc)
			return nil
		},
	)}
}

func listen(
	channel *rabbitmq.Channel,
	queue *amqp.Queue,
	onMessage func(*amqp.Delivery) error,
	lc fx.Lifecycle) {
	Logger.Infof("Listening queue %v", queue.Name)
	go func() {
		var deliveries <-chan amqp.Delivery
		var err error
		deliveries, err = channel.Consume(queue.Name, "", false, false, false, false, nil)
		if err != nil {
			Logger.Warnf("Unable to connect to queue %v, restarting. error %v", queue.Name, err)
			Logger.Panic(err)
		} else {
			Logger.Infof("Successfully connected to queue %v", queue.Name)
		}

		for msg := range deliveries {
			func() {
				defer func() {
					if err := recover(); err != nil {
						Logger.Errorf("In processing queue %v panic recovered: %v", queue.Name, err)
					}
				}()

				err := onMessage(&msg)
				if err != nil {
					Logger.Errorf("In processing queue %v error: %v", queue.Name, err)
				}
				err = msg.Ack(false)
				if err != nil {
					Logger.Errorf("In acking delivery for queue %v error: %v", queue.Name, err)
				}
			}()
		}
	}()
}
