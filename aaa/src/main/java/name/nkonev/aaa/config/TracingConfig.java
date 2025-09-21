package name.nkonev.aaa.config;

import io.micrometer.tracing.otel.bridge.OtelBaggageManager;
import io.micrometer.tracing.otel.bridge.OtelCurrentTraceContext;
import io.micrometer.tracing.otel.propagation.BaggageTextMapPropagator;
import io.opentelemetry.context.propagation.ContextPropagators;
import io.opentelemetry.context.propagation.TextMapPropagator;
import io.opentelemetry.extension.trace.propagation.JaegerPropagator;
import io.opentelemetry.sdk.logs.export.BatchLogRecordProcessor;
import io.opentelemetry.sdk.trace.export.BatchSpanProcessor;
import jakarta.annotation.PreDestroy;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.actuate.autoconfigure.tracing.TracingProperties;
import org.springframework.boot.autoconfigure.amqp.RabbitTemplateCustomizer;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.Collections;
import java.util.List;
import java.util.concurrent.TimeUnit;

@Configuration
public class TracingConfig {

    @Autowired
    private BatchSpanProcessor batchSpanProcessor;

    @Autowired
    private BatchLogRecordProcessor batchLogRecordProcessor;

    @Bean
    public JaegerPropagator jaegerPropagator() {
        return JaegerPropagator.getInstance();
    }

    @Bean
    TextMapPropagator jaegerTextMapPropagatorWithBaggage(TracingProperties tracingProperties, OtelCurrentTraceContext otelCurrentTraceContext, JaegerPropagator jaegerPropagator) {
        List<String> remoteFields = tracingProperties.getBaggage().getRemoteFields();
        BaggageTextMapPropagator baggagePropagator = new BaggageTextMapPropagator(remoteFields,
            new OtelBaggageManager(otelCurrentTraceContext, remoteFields, Collections.emptyList()));

        return TextMapPropagator.composite(baggagePropagator, jaegerPropagator);
    }

    // overrides OpenTelemetryAutoConfiguration.otelContextPropagators()
    @Bean
    ContextPropagators otelContextPropagators(TextMapPropagator jaegerTextMapPropagatorWithBaggage) {
        return ContextPropagators.create(jaegerTextMapPropagatorWithBaggage);
    }

    @PreDestroy
    public void pd() {
        // stop dangling threads (visible in tests)
        batchSpanProcessor.forceFlush().join(1, TimeUnit.MILLISECONDS).succeed().whenComplete(() -> {
                batchSpanProcessor.shutdown().join(1, TimeUnit.MILLISECONDS);
        }).succeed();

        batchLogRecordProcessor.forceFlush().join(1, TimeUnit.MILLISECONDS).succeed().whenComplete(() -> {
            batchLogRecordProcessor.shutdown().join(1, TimeUnit.MILLISECONDS);
        }).succeed();
    }

    @Bean
    public RabbitTemplateCustomizer rabbitTemplateTracingCustomizer() {
        return rabbitTemplate -> rabbitTemplate.setObservationEnabled(true);
    }
}
