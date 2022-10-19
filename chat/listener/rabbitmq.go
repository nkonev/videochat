package listener

import (
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	"go.uber.org/fx"
	. "nkonev.name/chat/logger"
	myRabbit "nkonev.name/chat/rabbitmq"
)

const aaaEventsQueue = "aaa-events"
const videoNotificationsQueue = "video-notifications"
const videoInviteQueue = "video-invite"
const videoDialStatusQueue = "video-dial-statuses"

type AaaEventsQueue struct{ *amqp.Queue }
type VideoInviteQueue struct{ *amqp.Queue }
type VideoDialStatusQueue struct{ *amqp.Queue }

type AaaEventsChannel struct{ *rabbitmq.Channel }
type VideoInviteChannel struct{ *rabbitmq.Channel }
type VideoDialStatusChannel struct{ *rabbitmq.Channel }

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

func CreateAaaChannel(connection *rabbitmq.Connection) AaaEventsChannel {
	return AaaEventsChannel{myRabbit.CreateRabbitMqChannel(connection)}
}

func CreateVideoInviteChannel(connection *rabbitmq.Connection) VideoInviteChannel {
	return VideoInviteChannel{myRabbit.CreateRabbitMqChannel(connection)}
}

func CreateVideoDialStatusChannel(connection *rabbitmq.Connection) VideoDialStatusChannel {
	return VideoDialStatusChannel{myRabbit.CreateRabbitMqChannel(connection)}
}

func CreateAaaQueue(consumeCh AaaEventsChannel) AaaEventsQueue {
	return AaaEventsQueue{create(aaaEventsQueue, consumeCh.Channel)}
}

func CreateVideoInviteQueue(consumeCh VideoInviteChannel) VideoInviteQueue {
	return VideoInviteQueue{create(videoInviteQueue, consumeCh.Channel)}
}

func CreateVideoDialStatusQueue(consumeCh VideoDialStatusChannel) VideoDialStatusQueue {
	return VideoDialStatusQueue{create(videoDialStatusQueue, consumeCh.Channel)}
}

//func CreateFanoutNotificationsQueue(consumeCh FanoutNotificationsChannel) FanoutNotificationsQueue {
//	fanoutQueue := createAndBind("", "", producer.DefaultFanoutExchange, consumeCh.Channel)
//	fanoutNotificationsQueueName = fanoutQueue.Name
//	return FanoutNotificationsQueue{fanoutQueue}
//}

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
			onMessage(&msg)
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

func ListenVideoInviteQueue(
	channel VideoInviteChannel,
	queue VideoInviteQueue,
	onMessage VideoInviteListener,
	lc fx.Lifecycle) {

	listen(channel.Channel, queue.Queue, onMessage, lc)
}

func ListenVideoDialStatusQueue(
	channel VideoDialStatusChannel,
	queue VideoDialStatusQueue,
	onMessage VideoDialStatusListener,
	lc fx.Lifecycle) {

	listen(channel.Channel, queue.Queue, onMessage, lc)
}
