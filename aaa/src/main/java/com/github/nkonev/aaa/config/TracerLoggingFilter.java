package com.github.nkonev.aaa.config;

import io.opentracing.Span;
import io.opentracing.Tracer;
import org.slf4j.MDC;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.core.annotation.Order;
import org.springframework.stereotype.Component;

import javax.servlet.*;
import java.io.IOException;

@Order(-2147483648)
@Component
public class TracerLoggingFilter implements Filter {
    @Autowired
    private Tracer tracer;

    @Override
    public void doFilter(ServletRequest servletRequest, ServletResponse servletResponse, FilterChain filterChain) throws IOException, ServletException {
        Span span = tracer.activeSpan();
        final String traceIdName = "traceId";
        final String spanIdName = "spanId";
        if (span != null) {
            String traceId = leadingZeros(span.context().toTraceId(), 32); // see at opentracing.jaeger.enable128-bit-traces
            String spanId = leadingZeros(span.context().toSpanId(), 16);
            MDC.put(traceIdName, traceId);
            MDC.put(spanIdName, spanId);
        }
        try {
            filterChain.doFilter(servletRequest, servletResponse);
        } finally {
            MDC.remove(traceIdName);
            MDC.remove(spanIdName);
        }
    }

    public String leadingZeros(String s, int length) {
        if (s.length() >= length) return s;
        else return String.format("%0" + (length-s.length()) + "d%s", 0, s);
    }
}
