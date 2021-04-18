package listener

import (
	"fmt"
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"go.uber.org/fx"
	"math/rand"
	. "nkonev.name/chat/logger"
	"time"
)

const aaaEventsQueue = "aaa-events"
const videoNotificationsQueue = "video-notifications"

func CreateRabbitMqConnection() *rabbitmq.Channel{
	rabbitmq.Debug = true

	conn, err := rabbitmq.Dial(viper.GetString("rabbitmq.url"))
	if err != nil {
		Logger.Panic(err)
	}

	consumeCh, err := conn.Channel()
	if err != nil {
		Logger.Panic(err)
	}
	return consumeCh
}

type AaaEventsQueue amqp.Queue
type VideoNotificationsQueue amqp.Queue

func create(name string, consumeCh *rabbitmq.Channel) *amqp.Queue {
	var err error
	var q amqp.Queue
	const maxRetries = 60
	var i = 0
	for ; i < maxRetries; i++ {
		q, err = consumeCh.QueueDeclare(
			name, // name
			true,   // durable
			false,   // delete when unused
			false,   // exclusive
			false,   // no-wait
			nil,     // arguments
		)
		if err != nil {
			Logger.Warnf("Unable to declare to queue %v, restarting. error %v", name, err)
		} else {
			Logger.Infof("Successfully declared queue %v", name)
			break
		}
		duration, _ := time.ParseDuration("1s")
		time.Sleep(duration)
	}

	if i == maxRetries {
		Logger.Errorf("Unable to declare queue %v after %v retries", name, maxRetries)
		Logger.Panic(err)
	}
	return &q
}

func CreateAaaQueue(consumeCh *rabbitmq.Channel) *AaaEventsQueue{
	q := create(aaaEventsQueue, consumeCh)
	var queue  AaaEventsQueue = AaaEventsQueue(*q)
	return &queue
}

func CreateVideoQueue(consumeCh *rabbitmq.Channel) *VideoNotificationsQueue{
	q := create(videoNotificationsQueue, consumeCh)
	var queue  VideoNotificationsQueue = VideoNotificationsQueue(*q)
	return &queue
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
		const maxRetries = 60
		var i = 0
		for ; i < maxRetries; i++ {
			rands := fmt.Sprintf("%v", rand.Intn(10000))
			deliveries, err = channel.Consume(queue.Name, "chat-"+queue.Name+"-consumer-"+rands, false, false, false, false, nil)
			if err != nil {
				Logger.Warnf("Unable to connect to queue %v, restarting. error %v", queue.Name, err)
			} else {
				Logger.Infof("Successfully connected to queue %v", queue.Name)
				break
			}
			duration, _ := time.ParseDuration("1s")
			time.Sleep(duration)
		}
		if i == maxRetries {
			Logger.Errorf("Unable to connect to queue %v after %v retries", queue.Name, maxRetries)
			Logger.Panic(err)
		}

		for msg := range deliveries {
			onMessage(msg.Body)
			msg.Ack(true)
		}
	}()
}

func ListenAaaQueue(
	channel *rabbitmq.Channel,
	queue *AaaEventsQueue,
	onMessage AaaUserProfileUpdateListener,
	lc fx.Lifecycle) {

	var typedQueue = *queue
	amqpQueue := amqp.Queue(typedQueue)

	var targetFunction func(data []byte) error = onMessage

	listen(channel, &amqpQueue, targetFunction, lc)
}

func ListenVideoQueue(
	channel *rabbitmq.Channel,
	queue *VideoNotificationsQueue,
	onMessage VideoListener,
	lc fx.Lifecycle) {

	var typedQueue = *queue
	amqpQueue := amqp.Queue(typedQueue)

	var targetFunction func(data []byte) error = onMessage

	listen(channel, &amqpQueue, targetFunction, lc)
}