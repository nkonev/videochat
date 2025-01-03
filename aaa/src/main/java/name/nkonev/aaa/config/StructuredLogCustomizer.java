package name.nkonev.aaa.config;

import ch.qos.logback.classic.spi.ILoggingEvent;
import org.springframework.boot.json.JsonWriter;
import org.springframework.boot.logging.structured.StructuredLoggingJsonMembersCustomizer;

public class StructuredLogCustomizer implements StructuredLoggingJsonMembersCustomizer<ILoggingEvent> {
    @Override
    public void customize(JsonWriter.Members<ILoggingEvent> members) {
            members.applyingNameProcessor((path, existingName) -> {
                if ("logger_name".equals(path.name())) {
                    return "logger";
                } if ("thread_name".equals(path.name())) {
                    return "thread";
                } if ("traceId".equals(path.name())) {
                    return "trace_id";
                } if ("spanId".equals(path.name())) {
                    return "span_id";
                } else {
                    return existingName;
                }
            });
            members.applyingPathFilter(memberPath -> {
                return "level_value".equals(memberPath.name()) || "@version".equals(memberPath.name());
            });
            members.applyingValueProcessor((path, value) -> {
                if ("level".equals(path.name())) {
                    return String.valueOf(value).toLowerCase();
                } else {
                    return value;
                }
            });
            members.add("service", "aaa");
    }
}
