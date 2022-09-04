module nkonev.name/video

require (
	github.com/araddon/dateparse v0.0.0-20200409225146-d820a6159ab1
	github.com/beliyav/go-amqp-reconnect v0.0.0-20200817192340-82ef0f85c3cc
	github.com/ehsaniara/gointerlock v1.1.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/google/uuid v1.3.0
	github.com/labstack/echo/v4 v4.7.2
	github.com/livekit/protocol v1.0.2-0.20220817073830-613285ea6f32
	github.com/livekit/server-sdk-go v0.10.5
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.7.0
	github.com/streadway/amqp v0.0.0-20190827072141-edfb9018d271
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho v0.32.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.32.0
	go.opentelemetry.io/contrib/propagators/jaeger v1.7.0
	go.opentelemetry.io/otel v1.7.0
	go.opentelemetry.io/otel/exporters/jaeger v1.7.0
	go.opentelemetry.io/otel/sdk v1.7.0
	go.opentelemetry.io/otel/trace v1.7.0
	go.uber.org/fx v1.12.0
	gopkg.in/ini.v1 v1.57.0 // indirect
)

go 1.16
