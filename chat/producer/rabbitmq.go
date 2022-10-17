package producer

import (
	"encoding/json"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
	. "nkonev.name/chat/logger"
	myRabbitmq "nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/type_registry"
	"time"
)

const AsyncEventsFanoutExchange = "async-events-exchange"

func (rp *RabbitFanoutNotificationsPublisher) Publish(aDto interface{}) error {
	aType := rp.typeRegistry.GetType(aDto)

	bytea, err := json.Marshal(aDto)
	if err != nil {
		Logger.Error(err, "Failed during marshal dto")
		return err
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Transient,
		Timestamp:    time.Now(),
		ContentType:  "application/json",
		Body:         bytea,
		Type:         aType,
	}

	if err := rp.channel.Publish(AsyncEventsFanoutExchange, "", false, false, msg); err != nil {
		Logger.Error(err, "Error during publishing dto")
		return err
	} else {
		return nil
	}
}

type RabbitFanoutNotificationsPublisher struct {
	channel      *rabbitmq.Channel
	typeRegistry *type_registry.TypeRegistryInstance
}

func NewRabbitNotificationsPublisher(connection *rabbitmq.Connection, typeRegistry *type_registry.TypeRegistryInstance) *RabbitFanoutNotificationsPublisher {
	return &RabbitFanoutNotificationsPublisher{
		channel:      myRabbitmq.CreateRabbitMqChannel(connection),
		typeRegistry: typeRegistry,
	}
}
