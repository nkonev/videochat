package com.github.nkonev.aaa.config;

import io.micrometer.tracing.otel.bridge.OtelBaggageManager;
import io.micrometer.tracing.otel.bridge.OtelCurrentTraceContext;
import io.micrometer.tracing.otel.propagation.BaggageTextMapPropagator;
import io.opentelemetry.context.propagation.ContextPropagators;
import io.opentelemetry.context.propagation.TextMapPropagator;
import io.opentelemetry.extension.trace.propagation.JaegerPropagator;
import org.springframework.boot.actuate.autoconfigure.tracing.TracingProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.Collections;
import java.util.List;

@Configuration
public class TracingConfig {

    @Bean
    TextMapPropagator jaegerTextMapPropagatorWithBaggage(TracingProperties tracingProperties, OtelCurrentTraceContext otelCurrentTraceContext) {
        List<String> remoteFields = tracingProperties.getBaggage().getRemoteFields();
        BaggageTextMapPropagator baggagePropagator = new BaggageTextMapPropagator(remoteFields,
            new OtelBaggageManager(otelCurrentTraceContext, remoteFields, Collections.emptyList()));

        return TextMapPropagator.composite(baggagePropagator, JaegerPropagator.getInstance());
    }

    // overrides OpenTelemetryAutoConfiguration.otelContextPropagators()
    @Bean
    ContextPropagators otelContextPropagators(TextMapPropagator jaegerTextMapPropagatorWithBaggage) {
        return ContextPropagators.create(jaegerTextMapPropagatorWithBaggage);
    }

}
