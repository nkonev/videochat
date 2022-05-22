module nkonev.name/video

require (
	github.com/araddon/dateparse v0.0.0-20200409225146-d820a6159ab1
	github.com/labstack/echo/v4 v4.7.2
	github.com/livekit/protocol v0.13.2
	github.com/livekit/server-sdk-go v0.10.0
	github.com/minio/minio-go/v7 v7.0.11
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.7.0
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho v0.32.0
	go.opentelemetry.io/contrib/propagators/jaeger v1.7.0
	go.opentelemetry.io/otel v1.7.0
	go.opentelemetry.io/otel/exporters/jaeger v1.7.0
	go.opentelemetry.io/otel/sdk v1.7.0
	go.opentelemetry.io/otel/trace v1.7.0
	go.uber.org/fx v1.12.0
)

go 1.16
