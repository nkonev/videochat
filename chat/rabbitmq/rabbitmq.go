package rabbitmq

import (
	"context"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	. "nkonev.name/chat/logger"
)

func CreateRabbitMqConnection() *rabbitmq.Connection {
	rabbitmq.Debug = true

	conn, err := rabbitmq.Dial(viper.GetString("rabbitmq.url"))
	if err != nil {
		Logger.Panic(err)
	}
	return conn
}

func CreateRabbitMqChannel(connection *rabbitmq.Connection) *rabbitmq.Channel {
	consumeCh, err := connection.Channel(nil)
	if err != nil {
		Logger.Panic(err)
	}
	return consumeCh
}

func CreateRabbitMqChannelWithCallback(connection *rabbitmq.Connection, clbFunc rabbitmq.ChannelCallbackFunc) *rabbitmq.Channel {
	consumeCh, err := connection.Channel(clbFunc)
	if err != nil {
		Logger.Panic(err)
	}
	return consumeCh
}

type AmqpHeadersCarrier map[string]interface{}

func (a AmqpHeadersCarrier) Get(key string) string {
	v, ok := a[key]
	if !ok {
		return ""
	}
	return v.(string)
}

func (a AmqpHeadersCarrier) Set(key string, value string) {
	a[key] = value
}

func (a AmqpHeadersCarrier) Keys() []string {
	i := 0
	r := make([]string, len(a))

	for k := range a {
		r[i] = k
		i++
	}

	return r
}

// InjectAMQPHeaders injects the tracing from the context into the header map
func InjectAMQPHeaders(ctx context.Context) map[string]interface{} {
	h := make(AmqpHeadersCarrier)
	otel.GetTextMapPropagator().Inject(ctx, h)
	return h
}

// ExtractAMQPHeaders extracts the tracing from the header and puts it into the context
func ExtractAMQPHeaders(ctx context.Context, headers map[string]interface{}) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, AmqpHeadersCarrier(headers))
}
