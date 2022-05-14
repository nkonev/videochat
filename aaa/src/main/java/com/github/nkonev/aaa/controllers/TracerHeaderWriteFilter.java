package com.github.nkonev.aaa.controllers;

import org.springframework.cloud.sleuth.Tracer;
import org.springframework.core.annotation.Order;
import org.springframework.stereotype.Component;

import javax.servlet.FilterChain;
import javax.servlet.ServletException;
import javax.servlet.http.HttpFilter;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.io.IOException;

@Order(-2147483648)
@Component
public class TracerHeaderWriteFilter extends HttpFilter {
    private final Tracer tracer;

    private static final String EXTERNAL_TRACE_ID_HEADER = "trace-id";

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
