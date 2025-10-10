package otel

import (
	"context"
	jaegerPropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"nkonev.name/chat/app"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"
)

func ConfigureTracePropagator() propagation.TextMapPropagator {
	return jaegerPropagator.Jaeger{}
}

func ConfigureTraceProvider(
	lgr *logger.LoggerWrapper,
	propagator propagation.TextMapPropagator,
	exporter *otlptrace.Exporter,
	lc fx.Lifecycle,
) *sdktrace.TracerProvider {
	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(app.TRACE_RESOURCE),
	)
	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(batchSpanProcessor),
		sdktrace.WithResource(resources),
	)
	otel.SetTracerProvider(tp)

	// register jaeger propagator
	otel.SetTextMapPropagator(propagator)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Info("Stopping trace provider")
			if err := tp.Shutdown(context.Background()); err != nil {
				lgr.Error("Error shutting trace provider", logger.AttributeError, err)
			}
			return nil
		},
	})
	return tp
}

func ConfigureTraceExporter(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	lc fx.Lifecycle,
) (*otlptrace.Exporter, error) {
	traceExporterConn, err := grpc.DialContext(context.Background(), cfg.Otlp.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	exporter, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithGRPCConn(traceExporterConn))
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Info("Stopping trace exporter")

			if err := exporter.Shutdown(ctx); err != nil {
				lgr.Error("Error shutting down trace exporter", logger.AttributeError, err)
			}

			if err := traceExporterConn.Close(); err != nil {
				lgr.Error("Error shutting down trace exporter connection", logger.AttributeError, err)
			}
			return nil
		},
	})

	return exporter, err
}
