package listener

import (
	"encoding/json"
	"fmt"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/lucsky/cuid"
	"go.uber.org/atomic"
	"nkonev.name/video/handlers"
	myRabbitmq "nkonev.name/video/rabbitmq"
)

type KickUserDto struct {
	ChatId int64 `json:"chatId"`
	UserId int64 `json:"userId"`
}

func createVideoListener(h *handlers.ExtendedService) myRabbitmq.VideoListenerFunction {
	return func(data []byte) error {
		var bindTo = new(KickUserDto)
		err := json.Unmarshal(data, &bindTo)
		if err != nil {
			logger.Error(err, "Error during deserialize KickUserDto")
			return err
		}
		logger.Info("Deserialized kick message", "chatId", bindTo.ChatId, "userId", bindTo.UserId)
		if err := h.KickUser(bindTo.ChatId, bindTo.UserId); err != nil {
			logger.Error(err, "Error during kicking user", "chatId", bindTo.ChatId, "userId", bindTo.UserId)
			return err
		}

		return nil
	}
}

type VideoListenerService struct {
	channel *rabbitmq.Channel
	listenerFunction myRabbitmq.VideoListenerFunction
}


func NewVideoListener(h *handlers.ExtendedService, connection *rabbitmq.Connection) *VideoListenerService {
	listener := createVideoListener(h)
	initialized := atomic.NewBool(false)
	queueNameText := fmt.Sprintf("video-kick-%v", cuid.New())
	channel := myRabbitmq.CreateRabbitMqChannelWithRecreate(connection, func(argChannel *rabbitmq.Channel) (error) {
		createFanoutExchange(videoKickExchange, argChannel)
		amqpQueue := createQueue(argChannel, queueNameText)
		bindQueueToExchange(videoKickExchange, amqpQueue, argChannel)

		if !initialized.Load() {
			listenQueue(argChannel, amqpQueue, listener)
			initialized.Store(true)
		}
		return nil
	})
	r := &VideoListenerService{
		channel: channel,
		listenerFunction: listener,
	}

	return r
}
