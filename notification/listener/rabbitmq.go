package listener

import (
	"context"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"go.uber.org/fx"
	"nkonev.name/notification/logger"
	myRabbit "nkonev.name/notification/rabbitmq"
)

const NotificationsExchange = "notifications-exchange"

type FanoutNotificationsChannel struct{ *rabbitmq.Channel }

func create(lgr *logger.Logger, name string, consumeCh *rabbitmq.Channel) *amqp.Queue {
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
		lgr.Warnf("Unable to declare to queue %v, restarting. error %v", name, err)
		lgr.Panic(err)
	}
	return &q
}

func createAndBind(lgr *logger.Logger, name string, key string, exchange string, consumeCh *rabbitmq.Channel) *amqp.Queue {
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
		lgr.Warnf("Unable to declare to queue %v, restarting. error %v", name, err)
		lgr.Panic(err)
	}
	err = consumeCh.QueueBind(q.Name, key, exchange, false, nil)
	if err != nil {
		lgr.Warnf("Unable to bind to queue %v, restarting. error %v", name, err)
		lgr.Panic(err)
	}
	return &q
}

func CreateNotificationsChannel(lgr *logger.Logger, connection *rabbitmq.Connection, onMessage NotificationsListener, lc fx.Lifecycle) FanoutNotificationsChannel {
	var queueName = "notifications"

	return FanoutNotificationsChannel{myRabbit.CreateRabbitMqChannelWithCallback(
		lgr,
		connection,
		func(channel *rabbitmq.Channel) error {
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					lgr.Infof("Stopping queue listening '%v'", queueName)
					return channel.Close()
				},
			})

			err := channel.ExchangeDeclare(NotificationsExchange, "direct", true, false, false, false, nil)
			if err != nil {
				return err
			}

			aQueue := createAndBind(lgr, queueName, "", NotificationsExchange, channel)
			listen(lgr, channel, aQueue, onMessage, lc)
			return nil
		},
	)}
}

func listen(
	lgr *logger.Logger,
	channel *rabbitmq.Channel,
	queue *amqp.Queue,
	onMessage func(*amqp.Delivery) error,
	lc fx.Lifecycle) {
	lgr.Infof("Listening queue %v", queue.Name)
	go func() {
		var deliveries <-chan amqp.Delivery
		var err error
		deliveries, err = channel.Consume(queue.Name, "", false, false, false, false, nil)
		if err != nil {
			lgr.Warnf("Unable to connect to queue %v, restarting. error %v", queue.Name, err)
			lgr.Panic(err)
		} else {
			lgr.Infof("Successfully connected to queue %v", queue.Name)
		}

		for msg := range deliveries {
			func() {
				defer func() {
					if err := recover(); err != nil {
						lgr.Errorf("In processing queue %v panic recovered: %v", queue.Name, err)
					}
				}()

				err := onMessage(&msg)
				if err != nil {
					lgr.Errorf("In processing queue %v error: %v", queue.Name, err)
				}
				err = msg.Ack(false)
				if err != nil {
					lgr.Errorf("In acking delivery for queue %v error: %v", queue.Name, err)
				}
			}()
		}
	}()
}
