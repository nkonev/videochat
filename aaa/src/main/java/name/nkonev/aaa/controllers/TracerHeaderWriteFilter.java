package name.nkonev.aaa.controllers;

import io.micrometer.tracing.Tracer;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpFilter;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.stereotype.Component;

import java.io.IOException;

@Order(Ordered.HIGHEST_PRECEDENCE + 2)
@Component
public class TracerHeaderWriteFilter extends HttpFilter {
    private final Tracer tracer;

    public static final String EXTERNAL_TRACE_ID_HEADER = "trace-id";

    public TracerHeaderWriteFilter(Tracer tracer) {
        this.tracer = tracer;
    }

    @Override
    public void doFilter(HttpServletRequest request, HttpServletResponse response,
                         FilterChain chain) throws IOException, ServletException {
        var currentSpan = this.tracer.currentSpan();
        if (currentSpan == null) {
            chain.doFilter(request, response);
        } else {
            var context = currentSpan.context();
            if (context != null) {
                var traceId = context.traceId();
                response.setHeader(EXTERNAL_TRACE_ID_HEADER, traceId);
            }
            chain.doFilter(request, response);
        }
    }
}
