package listener

import (
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"go.uber.org/fx"
	. "nkonev.name/chat/logger"
	myRabbit "nkonev.name/chat/rabbitmq"
)

const aaaEventsQueue = "aaa-events"
const videoNotificationsQueue = "video-notifications"


type AaaEventsQueue struct {*amqp.Queue}
type VideoNotificationsQueue struct {*amqp.Queue}

type AaaEventsChannel struct {*rabbitmq.Channel}
type VideoNotificationsChannel struct {*rabbitmq.Channel}

func create(name string, consumeCh *rabbitmq.Channel) *amqp.Queue {
	var err error
	var q amqp.Queue
	q, err = consumeCh.QueueDeclare(
		name, // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		Logger.Warnf("Unable to declare to queue %v, restarting. error %v", name, err)
		Logger.Panic(err)
	}
	return &q
}

func CreateAaaChannel(connection *rabbitmq.Connection) AaaEventsChannel{
	return AaaEventsChannel{myRabbit.CreateRabbitMqChannel(connection)}
}

func CreateVideoChannel(connection *rabbitmq.Connection) VideoNotificationsChannel{
	return VideoNotificationsChannel{myRabbit.CreateRabbitMqChannel(connection)}
}

func CreateAaaQueue(consumeCh AaaEventsChannel) AaaEventsQueue{
	return AaaEventsQueue{create(aaaEventsQueue, consumeCh.Channel)}
}

func CreateVideoQueue(consumeCh VideoNotificationsChannel) VideoNotificationsQueue{
	return VideoNotificationsQueue{create(videoNotificationsQueue, consumeCh.Channel)}
}

func listen(
	channel *rabbitmq.Channel,
	queue *amqp.Queue,
	onMessage func(data []byte) error,
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
			onMessage(msg.Body)
			msg.Ack(true)
		}
	}()
}

func ListenAaaQueue(
	channel AaaEventsChannel,
	queue AaaEventsQueue,
	onMessage AaaUserProfileUpdateListener,
	lc fx.Lifecycle) {

	listen(channel.Channel, queue.Queue, onMessage, lc)
}

func ListenVideoQueue(
	channel VideoNotificationsChannel,
	queue VideoNotificationsQueue,
	onMessage VideoListener,
	lc fx.Lifecycle) {

	listen(channel.Channel, queue.Queue, onMessage, lc)
}