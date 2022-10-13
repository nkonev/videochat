package producer

import (
	"encoding/json"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	. "nkonev.name/chat/logger"
	myRabbitmq "nkonev.name/chat/rabbitmq"
	"reflect"
	"time"
)

const FanoutNotificationsQueue = "chat-fanout-notifications"

func (rp *RabbitFanoutNotificationsPublisher) Publish(aDto interface{}) error {
	bytea, err := json.Marshal(aDto)
	if err != nil {
		Logger.Error(err, "Failed during marshal dto")
		return err
	}

	aType := reflect.TypeOf(aDto).Name()
	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         aType,
	}

	if err := rp.channel.Publish("", FanoutNotificationsQueue, false, false, msg); err != nil {
		Logger.Error(err, "Error during publishing dto")
		return err
	} else {
		return nil
	}
}

type RabbitFanoutNotificationsPublisher struct {
	channel *rabbitmq.Channel
}

func NewRabbitNotificationsPublisher(connection *rabbitmq.Connection) *RabbitFanoutNotificationsPublisher {
	return &RabbitFanoutNotificationsPublisher{
		channel: myRabbitmq.CreateRabbitMqChannel(connection),
	}
}
