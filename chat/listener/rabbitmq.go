package listener

import (
	"context"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	myRabbit "nkonev.name/chat/rabbitmq"

	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"go.uber.org/fx"
)

const testOutputQueueName = "chat-output-event-test"
const testNotificationQueueName = "chat-notification-event-test"
const aaaEventsQueue = "chat-aaa-profile-events"

type ChatChannel struct{ *rabbitmq.Channel }
type InternalEventChannel struct{ *rabbitmq.Channel }

type AaaEventsChannel struct{ *rabbitmq.Channel }
type AaaEventsQueue struct{ *amqp.Queue }

func create(lgr *logger.LoggerWrapper, name string, consumeCh *rabbitmq.Channel) (*amqp.Queue, error) {
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
		lgr.Warn("Unable to declare to queue, restarting.", "queue", name, logger.AttributeError, err)
		return nil, err
	}
	return &q, nil
}

func createAndBind(lgr *logger.LoggerWrapper, name string, key string, exchange string, consumeCh *rabbitmq.Channel) (*amqp.Queue, error) {
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
		lgr.Warn("Unable to declare to queue, restarting.", "queue", name, logger.AttributeError, err)
		return nil, err
	}
	err = consumeCh.QueueBind(q.Name, key, exchange, false, nil)
	if err != nil {
		lgr.Warn("Unable to bind to queue, restarting.", "queue", name, logger.AttributeError, err)
		return nil, err
	}
	return &q, nil
}

func DeleteTestEventQueue(lgr *logger.LoggerWrapper, connection *rabbitmq.Connection) error {
	ch, err := myRabbit.CreateRabbitMqChannel(
		lgr,
		connection,
	)
	if err != nil {
		return err
	}

	lgr.Warn("Deleting test queue", "queue", testOutputQueueName)
	_, err = ch.QueueDelete(testOutputQueueName, false, false, false)
	if err != nil {
		lgr.Warn("An error during delete", "queue", testOutputQueueName, logger.AttributeError, err)
	}
	return nil
}

func CreateAndListenTestOutputEventChannel(lgr *logger.LoggerWrapper, connection *rabbitmq.Connection, onMessage TestOutputEventListener, lc fx.Lifecycle, sh fx.Shutdowner) (*ChatChannel, error) {

	ch, err := myRabbit.CreateRabbitMqChannelWithCallback(
		lgr,
		connection,
		func(channel *rabbitmq.Channel) error {
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					lgr.Info("Stopping queue listening", "queue", testOutputQueueName)
					return channel.Close()
				},
			})

			err := channel.ExchangeDeclare(producer.EventsFanoutExchange, "fanout", true, false, false, false, nil)
			if err != nil {
				return err
			}

			aQueue, err := createAndBind(lgr, testOutputQueueName, "", producer.EventsFanoutExchange, channel)
			if err != nil {
				return err
			}

			listen(lgr, channel, aQueue, onMessage, sh)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &ChatChannel{ch}, nil
}

func CreateAndListenTestNotificationEventChannel(lgr *logger.LoggerWrapper, connection *rabbitmq.Connection, onMessage TestNotificationEventListener, lc fx.Lifecycle, sh fx.Shutdowner) (*ChatChannel, error) {

	ch, err := myRabbit.CreateRabbitMqChannelWithCallback(
		lgr,
		connection,
		func(channel *rabbitmq.Channel) error {
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					lgr.Info("Stopping queue listening", "queue", testNotificationQueueName)
					return channel.Close()
				},
			})

			err := channel.ExchangeDeclare(producer.NotificationsFanoutExchange, "direct", true, false, false, false, nil)
			if err != nil {
				return err
			}

			aQueue, err := createAndBind(lgr, testNotificationQueueName, "", producer.NotificationsFanoutExchange, channel)
			if err != nil {
				return err
			}

			listen(lgr, channel, aQueue, onMessage, sh)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &ChatChannel{ch}, nil
}

func CreateAndListenInternalEventsChannel(
	lgr *logger.LoggerWrapper,
	connection *rabbitmq.Connection,
	onMessage InternalEventsListener,
	sh fx.Shutdowner,
	lc fx.Lifecycle,
) (*InternalEventChannel, error) {
	var internalQueueName = "chat-internal"

	ch, err := myRabbit.CreateRabbitMqChannelWithCallback(
		lgr,
		connection,
		func(channel *rabbitmq.Channel) error {
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					lgr.Info("Stopping queue listening", "queue", internalQueueName)
					return channel.Close()
				},
			})

			err := channel.ExchangeDeclare(producer.ChatInternalExchange, "direct", true, false, false, false, nil)
			if err != nil {
				return err
			}

			aQueue, err := createAndBind(lgr, internalQueueName, "", producer.ChatInternalExchange, channel)
			if err != nil {
				return err
			}

			listen(lgr, channel, aQueue, onMessage, sh)
			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return &InternalEventChannel{ch}, nil
}

func CreateAndListenAaaChannel(lgr *logger.LoggerWrapper, connection *rabbitmq.Connection, onMessage AaaUserProfileUpdateListener, lc fx.Lifecycle, sh fx.Shutdowner) (*AaaEventsChannel, error) {
	ch, err := myRabbit.CreateRabbitMqChannelWithCallback(
		lgr,
		connection,
		func(channel *rabbitmq.Channel) error {
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					lgr.Info("Stopping queue listening", "queue", aaaEventsQueue)
					return channel.Close()
				},
			})

			err := channel.ExchangeDeclare(producer.AaaEventsExchange, "fanout", true, false, false, false, nil)
			if err != nil {
				return err
			}

			aQueue, err := createAndBind(lgr, aaaEventsQueue, "", producer.AaaEventsExchange, channel)
			if err != nil {
				return err
			}

			listen(lgr, channel, aQueue, onMessage, sh)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &AaaEventsChannel{ch}, nil
}

func listen(
	lgr *logger.LoggerWrapper,
	channel *rabbitmq.Channel,
	queue *amqp.Queue,
	onMessage func(*amqp.Delivery) error,
	sh fx.Shutdowner) {
	lgr.Info("Listening queue", "queue", queue.Name)
	go func() {
		var deliveries <-chan amqp.Delivery
		var errOuter error
		deliveries, errOuter = channel.Consume(queue.Name, "", false, false, false, false, nil)
		if errOuter != nil {
			lgr.Error("Unable to connect to queue, restarting", "queue", queue.Name, logger.AttributeError, errOuter)
			sh.Shutdown()
			return
		} else {
			lgr.Info("Successfully connected to queue", "queue", queue.Name)
		}

		for msg := range deliveries {
			func() {
				defer func() {
					if err := recover(); err != nil {
						lgr.Error("In processing queue panic recovered", "queue", queue.Name, logger.AttributeError, err)
					}
				}()

				err := onMessage(&msg)
				if err != nil {
					lgr.Error("In processing queue error", "queue", queue.Name, logger.AttributeError, err)
				}
				err = msg.Ack(false)
				if err != nil {
					lgr.Error("In acking delivery for queue error", "queue", queue.Name, logger.AttributeError, err)
				}
			}()
		}
	}()
}
