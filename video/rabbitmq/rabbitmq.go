package rabbitmq

import (
	"context"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx"
	"nkonev.name/video/config"
	"nkonev.name/video/logger"
)

type VideoListenerFunction func(data []byte) error

func CreateRabbitMqConnection(lgr *logger.Logger, conf *config.ExtendedConfig, lc fx.Lifecycle) *rabbitmq.Connection {
	rabbitmq.Debug = conf.RabbitMqConfig.Debug

	conn, err := rabbitmq.Dial(conf.RabbitMqConfig.Url)
	if err != nil {
		lgr.Error(err, "Unable to connect to rabbitmq")
		panic(err)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Info("Closing rabbitmq connection")
			return conn.Close()
		},
	})

	return conn
}

func CreateRabbitMqChannelWithRecreate(
	lgr *logger.Logger,
	connection *rabbitmq.Connection,
	callback func(argChannel *rabbitmq.Channel) error,
) *rabbitmq.Channel {
	channel, err := connection.Channel(callback)
	if err != nil {
		lgr.Error(err, "Unable to create channel")
		panic(err)
	}
	return channel
}

func CreateRabbitMqChannel(
	lgr *logger.Logger,
	connection *rabbitmq.Connection,
) *rabbitmq.Channel {
	channel, err := connection.Channel(func(argChannel *rabbitmq.Channel) error {
		return nil
	})
	if err != nil {
		lgr.Error(err, "Unable to create channel")
		panic(err)
	}
	return channel
}

func CreateRabbitMqChannelWithCallback(lgr *logger.Logger, connection *rabbitmq.Connection, clbFunc rabbitmq.ChannelCallbackFunc) *rabbitmq.Channel {
	consumeCh, err := connection.Channel(clbFunc)
	if err != nil {
		lgr.Panic(err)
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
