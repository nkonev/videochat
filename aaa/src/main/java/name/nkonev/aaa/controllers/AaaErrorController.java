package name.nkonev.aaa.controllers;

import java.io.IOException;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.stream.Collectors;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.web.error.ErrorAttributeOptions;
import org.springframework.boot.webmvc.autoconfigure.error.AbstractErrorController;
import org.springframework.boot.webmvc.autoconfigure.error.ErrorViewResolver;
import org.springframework.boot.webmvc.error.ErrorAttributes;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.security.authorization.AuthorizationDeniedException;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.servlet.ModelAndView;

import com.fasterxml.jackson.databind.ObjectMapper;

import io.micrometer.tracing.Tracer;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.dto.AaaError;
import static name.nkonev.aaa.security.RESTAuthenticationFailureHandler.AAA_AUTH_FAILURE_KEY;
import static name.nkonev.aaa.security.RESTAuthenticationFailureHandler.AAA_AUTH_FAILURE_MESSAGE;
import static name.nkonev.aaa.utils.ServletUtils.getAcceptHeaderValues;
import static name.nkonev.aaa.utils.NullUtils.getToStringSafe;

/**
 * @see org.springframework.boot.webmvc.autoconfigure.error.BasicErrorController, it describes how to use both REST And ModelAndView handling depends on Accept header
 * @see "https://gist.github.com/jonikarppinen/662c38fb57a23de61c8b"
 */
@Controller
public class AaaErrorController extends AbstractErrorController {

    private static final String PATH = "/error";

    @Autowired
    private ObjectMapper objectMapper;

    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private Tracer tracer;

    private static final Logger LOGGER = LoggerFactory.getLogger(AaaErrorController.class);

    public AaaErrorController(ErrorAttributes errorAttributes, List<ErrorViewResolver> errorViewResolvers) {
        super(errorAttributes, errorViewResolvers);
    }

    private final Set<Class<?>> noErrorExceptions = Set.of(AuthorizationDeniedException.class);

    @RequestMapping(value = PATH)
    public ModelAndView error(HttpServletRequest request, HttpServletResponse response) throws ServletException, IOException {

        final List<String> acceptValues = getAcceptHeaderValues(request);

        Map<String, Object> errorAttributes = getCustomErrorAttributes(request);

        final var isAuthFailure = Boolean.TRUE.equals(request.getAttribute(AAA_AUTH_FAILURE_KEY));
        String message = getToStringSafe(()->errorAttributes.get("message"));
        if (isAuthFailure) {
            message = getToStringSafe(()->request.getAttribute(AAA_AUTH_FAILURE_MESSAGE));
        }

        if (noErrorExceptions.stream()
                .map(Class::getCanonicalName)
                .filter(Objects::nonNull)
                .anyMatch(se -> se.equals(errorAttributes.get("exception")))
            || isAuthFailure
        ) {
            LOGGER.debug("Message: {}, error: {}, exception: {}", message, errorAttributes.get("error"), errorAttributes.get("exception"));
        } else {
            LOGGER.error("Message: {}, error: {}, exception: {}, trace: {}", message, errorAttributes.get("error"), errorAttributes.get("exception"), errorAttributes.get("trace"));
        }

        if (acceptValues.contains(MediaType.APPLICATION_JSON_VALUE)) {
            response.setContentType(MediaType.APPLICATION_JSON_VALUE);
            try {
                if (aaaProperties.debugResponse()) {
                    objectMapper.writeValue(response.getWriter(), new AaaError(
                            response.getStatus(),
                            (String) errorAttributes.get("error"),
                            message,
                            errorAttributes.get("timestamp").toString(),
                            (String) errorAttributes.get("exception"),
                            (String) errorAttributes.get("trace"))
                    );
                } else {
                    objectMapper.writeValue(response.getWriter(), new AaaError(
                            response.getStatus(),
                            (String) errorAttributes.get("error"),
                            message,
                            errorAttributes.get("timestamp").toString()
                    ));
                }
            } catch (IOException e){
                LOGGER.error("IOException", e);
            }
            return null;
        } else {
            HttpStatus status = getStatus(request);
            Map<String, Object> m;
            if (aaaProperties.debugResponse()) {
                m = new HashMap<>(errorAttributes);
            } else {
                m = new HashMap<>(errorAttributes.entrySet().stream()
                        .filter(e -> !"trace".equals(e.getKey()))
                        .filter(e -> !"exception".equals(e.getKey()))
                        .collect(Collectors.toMap(Map.Entry::getKey, Map.Entry::getValue)));
            }
            var traceId = getTraceId();
            if (traceId != null) {
                m.put("traceId", traceId);
            }
            m.put("status", status.value());
            var model = Collections.unmodifiableMap(m);
            response.setStatus(status.value());
            // see ErrorMvcAutoConfiguration.StaticView
            ModelAndView modelAndView = resolveErrorView(request, response, status, model);
            return (modelAndView == null ? new ModelAndView("error", model) : modelAndView);
        }
    }

    private Map<String, Object> getCustomErrorAttributes(HttpServletRequest request) {
        return getErrorAttributes(request, ErrorAttributeOptions.of(ErrorAttributeOptions.Include.MESSAGE, ErrorAttributeOptions.Include.EXCEPTION, ErrorAttributeOptions.Include.STACK_TRACE));
    }

    private String getTraceId() {
        var currentSpan = this.tracer.currentSpan();
        if (currentSpan == null) {
            return null;
        } else {
            var context = currentSpan.context();
            if (context != null) {
                var traceId = context.traceId();
                return traceId;
            }
            return null;
        }
    }
}
