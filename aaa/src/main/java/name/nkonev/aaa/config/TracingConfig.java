package name.nkonev.aaa.config;

import io.micrometer.tracing.otel.bridge.OtelBaggageManager;
import io.micrometer.tracing.otel.bridge.OtelCurrentTraceContext;
import io.micrometer.tracing.otel.propagation.BaggageTextMapPropagator;
import io.opentelemetry.context.Context;
import io.opentelemetry.context.propagation.ContextPropagators;
import io.opentelemetry.context.propagation.TextMapPropagator;
import io.opentelemetry.context.propagation.TextMapSetter;
import io.opentelemetry.extension.trace.propagation.JaegerPropagator;
import io.opentelemetry.sdk.trace.export.BatchSpanProcessor;
import jakarta.annotation.PreDestroy;
import org.springframework.amqp.AmqpException;
import org.springframework.amqp.core.Message;
import org.springframework.amqp.core.MessagePostProcessor;
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
        batchSpanProcessor.forceFlush().join(1, TimeUnit.MILLISECONDS).succeed().whenComplete(() -> {
                batchSpanProcessor.shutdown().join(1, TimeUnit.MILLISECONDS);
        }).succeed();
    }

    @Bean
    public RabbitTemplateCustomizer rabbitTemplateTracingCustomizer(TextMapPropagator jaegerTextMapPropagatorWithBaggage) {
        return rabbitTemplate -> rabbitTemplate.addBeforePublishPostProcessors(new RabbitTemplateTracingMessagePostProcessor(jaegerTextMapPropagatorWithBaggage));
    }
}

class RabbitTemplateTracingMessagePostProcessor implements MessagePostProcessor, TextMapSetter<Message> {

    final TextMapPropagator textMapPropagator;

    RabbitTemplateTracingMessagePostProcessor(TextMapPropagator textMapPropagator) {
        this.textMapPropagator = textMapPropagator;
    }

    @Override
    public Message postProcessMessage(Message message) throws AmqpException {
        textMapPropagator.inject(Context.current(), message, this);
        return message;
    }

    @Override
    public void set(Message carrier, String key, String value) {
        if (carrier != null && key != null && value != null) {
            carrier.getMessageProperties().setHeader(key, value);
        }
    }
}
