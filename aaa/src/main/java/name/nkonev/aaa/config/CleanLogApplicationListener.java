package name.nkonev.aaa.config;

import org.springframework.boot.context.event.ApplicationEnvironmentPreparedEvent;
import org.springframework.boot.context.event.ApplicationFailedEvent;
import org.springframework.boot.context.event.ApplicationPreparedEvent;
import org.springframework.boot.context.event.ApplicationStartingEvent;
import org.springframework.boot.context.logging.LoggingApplicationListener;
import org.springframework.context.ApplicationEvent;
import org.springframework.context.event.ContextClosedEvent;
import org.springframework.context.event.GenericApplicationListener;
import org.springframework.core.ResolvableType;
import org.springframework.util.StringUtils;

import java.io.File;

public class CleanLogApplicationListener implements GenericApplicationListener {
    // Run before LoggingApplicationListener
    private static final int order = LoggingApplicationListener.DEFAULT_ORDER - 2;

    private static final Class<?>[] EVENT_TYPES = { ApplicationEnvironmentPreparedEvent.class };

    private volatile boolean processed = false;

    @Override
    public boolean supportsEventType(ResolvableType resolvableType) {
        return isAssignableFrom(resolvableType.getRawClass(), EVENT_TYPES);
    }

    private boolean isAssignableFrom(Class<?> type, Class<?>... supportedTypes) {
        if (type != null) {
            for (Class<?> supportedType : supportedTypes) {
                if (supportedType.isAssignableFrom(type)) {
                    return true;
                }
            }
        }
        return false;
    }

    @Override
    public void onApplicationEvent(ApplicationEvent event) {
        if (event instanceof ApplicationEnvironmentPreparedEvent environmentPreparedEvent) {
            onApplicationEnvironmentPreparedEvent(environmentPreparedEvent);
        }
    }

    private void onApplicationEnvironmentPreparedEvent(ApplicationEnvironmentPreparedEvent event) {
        var fn = event.getEnvironment().getProperty("logging.file.name");
        if (StringUtils.hasLength(fn) && !this.processed) {
            new File(fn).delete();
            this.processed = true;
        }
    }

    @Override
    public int getOrder() {
        return this.order;
    }
}
