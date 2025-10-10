package rabbitmq

import (
	"context"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"
)

func CreateRabbitMqConnection(lgr *logger.LoggerWrapper, cfg *config.AppConfig, lc fx.Lifecycle) (*rabbitmq.Connection, error) {
	rabbitmq.Debug = cfg.RabbitMQ.Debug

	conn, err := rabbitmq.Dial(cfg.RabbitMQ.Url)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Info("Closing rabbitmq connection")
			return conn.Close()
		},
	})

	return conn, nil
}

func CreateRabbitMqChannel(lgr *logger.LoggerWrapper, connection *rabbitmq.Connection) (*rabbitmq.Channel, error) {
	consumeCh, err := connection.Channel(nil)
	if err != nil {
		return nil, err
	}
	return consumeCh, nil
}

func CreateRabbitMqChannelWithCallback(lgr *logger.LoggerWrapper, connection *rabbitmq.Connection, clbFunc rabbitmq.ChannelCallbackFunc) (*rabbitmq.Channel, error) {
	consumeCh, err := connection.Channel(clbFunc)
	if err != nil {
		return nil, err
	}
	return consumeCh, nil
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
