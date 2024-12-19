package listener

import (
	"context"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"go.uber.org/fx"
	myRabbit "nkonev.name/video/rabbitmq"
)

const AaaEventsExchange = "aaa-profile-events-exchange"

const aaaEventsQueue = "video-aaa-profile-events"

type AaaEventsQueue struct{ *amqp.Queue }

type AaaEventsChannel struct{ *rabbitmq.Channel }

func create(lgr *log.Logger, name string, consumeCh *rabbitmq.Channel) *amqp.Queue {
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
	return &q
}

func createAndBind(lgr *log.Logger, name string, key string, exchange string, consumeCh *rabbitmq.Channel) *amqp.Queue {
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
func CreateAaaChannel(lgr *log.Logger, connection *rabbitmq.Connection, onMessage AaaUserProfileUpdateListener, lc fx.Lifecycle) *AaaEventsChannel {
	return &AaaEventsChannel{myRabbit.CreateRabbitMqChannelWithCallback(
		lgr,
		connection,
		func(channel *rabbitmq.Channel) error {
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					lgr.Infof("Stopping queue listening '%v'", aaaEventsQueue)
					return channel.Close()
				},
			})

			err := channel.ExchangeDeclare(AaaEventsExchange, "fanout", true, false, false, false, nil)
			if err != nil {
				return err
			}

			aQueue := createAndBind(lgr, aaaEventsQueue, "", AaaEventsExchange, channel)
			listen(lgr, channel, aQueue, onMessage, lc)
			return nil
		},
	)}
}

func CreateAaaQueue(lgr *log.Logger, consumeCh *AaaEventsChannel) *AaaEventsQueue {
	return &AaaEventsQueue{create(lgr, aaaEventsQueue, consumeCh.Channel)}
}

func listen(
	lgr *log.Logger,
	channel *rabbitmq.Channel,
	queue *amqp.Queue,
	onMessage func(*amqp.Delivery) error,
	lc fx.Lifecycle,
) {
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

func ListenAaaQueue(
	lgr *log.Logger,
	channel *AaaEventsChannel,
	queue *AaaEventsQueue,
	onMessage AaaUserProfileUpdateListener,
	lc fx.Lifecycle) {

	listen(lgr, channel.Channel, queue.Queue, onMessage, lc)
}
