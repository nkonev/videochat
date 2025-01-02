package name.nkonev.aaa.config;

import ch.qos.logback.classic.spi.ILoggingEvent;
import org.springframework.boot.json.JsonWriter;
import org.springframework.boot.logging.structured.StructuredLoggingJsonMembersCustomizer;

// see also org.springframework.boot.logging.logback.ElasticCommonSchemaStructuredLogFormatter
public class StructuredLogCustomizer implements StructuredLoggingJsonMembersCustomizer<ILoggingEvent> {
    @Override
    public void customize(JsonWriter.Members<ILoggingEvent> members) {
            members.applyingValueProcessor((path, value) -> {
                if ("log.level".equals(path.name())) {
                    return String.valueOf(value).toLowerCase();
                } else {
                    return value;
                }
            });

            members.applyingNameProcessor((path, existingName) -> {
                if ("log.logger".equals(path.name())) {
                    return "logger";
                } if ("process.thread.name".equals(path.name())) {
                    return "thread";
                } if ("process.pid".equals(path.name())) {
                    return "pid";
                } if ("traceId".equals(path.name())) {
                    return "trace_id";
                } if ("spanId".equals(path.name())) {
                    return "span_id";
                } if ("service.name".equals(path.name())) {
                    return "service";
                } if ("log.level".equals(path.name())) {
                    return "level";
                } if ("error.type".equals(path.name())) {
                    return "error_type";
                } if ("error.message".equals(path.name())) {
                    return "error_message";
                } if ("error.stack_trace".equals(path.name())) {
                    return "stack_trace";
                } else {
                    return existingName;
                }
            });
            members.applyingPathFilter(memberPath -> {
                return "ecs.version".equals(memberPath.name());
            });
    }
}
