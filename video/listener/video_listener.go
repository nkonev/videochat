package listener

import (
	"encoding/json"
	"fmt"
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	"nkonev.name/video/handlers"
	myRabbitmq "nkonev.name/video/rabbitmq"
)

type VideoListenerFunction func(data []byte) error

type KickUserDto struct {
	ChatId int64 `json:"chatId"`
	UserId int64 `json:"userId"`
}

func createVideoListener(h *handlers.HttpHandler) VideoListenerFunction {
	return func(data []byte) error {
		var bindTo = new(KickUserDto)
		err := json.Unmarshal(data, &bindTo)
		if err != nil {
			logger.Error(err, "Error during deserialize KickUserDto")
			return err
		}
		logger.Info("Deserialized kick message", "chatId", bindTo.ChatId, "userId", bindTo.UserId)
		if err := h.KickUser(fmt.Sprintf("%v", bindTo.ChatId), fmt.Sprintf("%v", bindTo.UserId)); err != nil {
			logger.Error(err, "Error during kicking user", "chatId", bindTo.ChatId, "userId", bindTo.UserId)
			return err
		}

		return nil
	}
}

type VideoListenerService struct {
	channel *rabbitmq.Channel
	listenerFunction VideoListenerFunction
}


func NewVideoListener(h *handlers.HttpHandler, connection *rabbitmq.Connection) *VideoListenerService {
	channel := myRabbitmq.CreateRabbitMqChannel(connection)
	listener := createVideoListener(h)

	return &VideoListenerService{
		channel: channel,
		listenerFunction: listener,
	}
}

func (r *VideoListenerService) ListenVideoKickQueue() {
	createFanoutExchange(videoKickExchange, r.channel)
	amqpQueue := createAnonymousQueue(r.channel)
	bindQueueToExchange(videoKickExchange, amqpQueue, r.channel)
	listenQueue(r.channel, amqpQueue, r.listenerFunction)
}