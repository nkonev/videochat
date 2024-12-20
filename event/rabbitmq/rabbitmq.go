package rabbitmq

import (
	"context"
	"encoding/json"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"nkonev.name/event/logger"
)

func CreateRabbitMqConnection(lgr *log.Logger) *rabbitmq.Connection {
	rabbitmq.Debug = true

	conn, err := rabbitmq.Dial(viper.GetString("rabbitmq.url"))
	if err != nil {
		lgr.Panic(err)
	}
	return conn
}

func CreateRabbitMqChannel(lgr *log.Logger, connection *rabbitmq.Connection) *rabbitmq.Channel {
	consumeCh, err := connection.Channel(nil)
	if err != nil {
		lgr.Panic(err)
	}
	return consumeCh
}

func CreateRabbitMqChannelWithCallback(lgr *log.Logger, connection *rabbitmq.Connection, clbFunc rabbitmq.ChannelCallbackFunc) *rabbitmq.Channel {
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

func SerializeValues(spanContext context.Context, lgr *log.Logger) string {
	carrier := propagation.MapCarrier{}
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(spanContext, carrier)
	marshal, err := json.Marshal(carrier)
	if err != nil {
		logger.GetLogEntry(spanContext, lgr).Infof("Unable to marshall")
		return ""
	}
	return string(marshal)
}

func DeserializeValues(spanContext context.Context, lgr *log.Logger, input string) context.Context {
	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier{}

	err := json.Unmarshal([]byte(input), &carrier)
	if err != nil {
		logger.GetLogEntry(spanContext, lgr).Infof("Unable to unmarshall")
		return context.Background()
	}

	return propagator.Extract(context.Background(), carrier)
}
