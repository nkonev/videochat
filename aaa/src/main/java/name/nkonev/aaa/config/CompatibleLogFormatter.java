package name.nkonev.aaa.config;

import ch.qos.logback.classic.pattern.ThrowableProxyConverter;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.classic.spi.IThrowableProxy;
import org.slf4j.event.KeyValuePair;
import org.springframework.boot.json.JsonWriter;
import org.springframework.boot.logging.structured.JsonWriterStructuredLogFormatter;
import org.springframework.boot.logging.structured.StructuredLoggingJsonMembersCustomizer;
import org.springframework.core.env.Environment;

import java.util.AbstractMap;
import java.util.Map;
import java.util.Objects;
import java.util.stream.Collectors;

// see also org.springframework.boot.logging.logback.ElasticCommonSchemaStructuredLogFormatter
public class CompatibleLogFormatter extends JsonWriterStructuredLogFormatter<ILoggingEvent> {

    private static final JsonWriter.PairExtractor<KeyValuePair> keyValuePairExtractor = JsonWriter.PairExtractor.of((pair) -> pair.key,
            (pair) -> pair.value);

    CompatibleLogFormatter(Environment environment, ThrowableProxyConverter throwableProxyConverter,
                                              StructuredLoggingJsonMembersCustomizer<?> customizer) {
        super((members) -> jsonMembers(environment, throwableProxyConverter, members), customizer);
    }

    private static void jsonMembers(Environment environment, ThrowableProxyConverter throwableProxyConverter,
                                    JsonWriter.Members<ILoggingEvent> members) {
        members.add("@timestamp", ILoggingEvent::getInstant);
        members.add("level", event -> {
            var ll = event.getLevel();
            if (ll != null) {
                return ll.toString().toLowerCase();
            } else {
                return ll;
            }
        });
        members.add("pid", environment.getProperty("spring.application.pid", Long.class)).when(Objects::nonNull);
        members.add("thread", ILoggingEvent::getThreadName);
        members.add("service", environment.getProperty("spring.application.name")).whenHasLength();
        members.add("logger", ILoggingEvent::getLoggerName);
        members.add("message", ILoggingEvent::getFormattedMessage);
        members.addMapEntries(event -> event.getMDCPropertyMap().entrySet().stream().map(entry -> {
            var key = entry.getKey();
            switch (key) {
                case "traceId" -> key = "trace_id";
                case "spanId" -> key = "span_id";
            }
            var value = entry.getValue();
            return new AbstractMap.SimpleEntry<>(key, value);
        }).collect(Collectors.toMap(AbstractMap.SimpleEntry::getKey, AbstractMap.SimpleEntry::getValue)));
        members.from(ILoggingEvent::getKeyValuePairs)
                .whenNotEmpty()
                .usingExtractedPairs(Iterable::forEach, keyValuePairExtractor);
        members.add().whenNotNull(ILoggingEvent::getThrowableProxy).usingMembers((throwableMembers) -> {
            throwableMembers.add("error_type", ILoggingEvent::getThrowableProxy).as(IThrowableProxy::getClassName);
            throwableMembers.add("error_message", ILoggingEvent::getThrowableProxy).as(IThrowableProxy::getMessage);
            throwableMembers.add("stack_trace", throwableProxyConverter::convert);
        });
    }

}
