package com.github.nkonev.blog.controllers;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.dto.BlogError;
import com.github.nkonev.blog.dto.BlogErrorWithDebug;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.web.servlet.error.AbstractErrorController;
import org.springframework.boot.autoconfigure.web.servlet.error.ErrorViewResolver;
import org.springframework.boot.web.servlet.error.ErrorAttributes;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;
import org.springframework.web.servlet.ModelAndView;

import javax.servlet.RequestDispatcher;
import javax.servlet.ServletException;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

/**
 * @see org.springframework.boot.autoconfigure.web.servlet.error.BasicErrorController, it describes how to use both REST And ModelAndView handling depends on Accept header
 * @see "https://gist.github.com/jonikarppinen/662c38fb57a23de61c8b"
 */
@Controller
public class BlogErrorController extends AbstractErrorController {

    @Value("${debugResponse:false}")
    private boolean debug;

    private static final String PATH = "/error";

    @Autowired
    private ObjectMapper objectMapper;

    private static final Logger LOGGER = LoggerFactory.getLogger(BlogErrorController.class);

    public BlogErrorController(ErrorAttributes errorAttributes, List<ErrorViewResolver> errorViewResolvers) {
        super(errorAttributes, errorViewResolvers);
    }

    @Override
    public String getErrorPath() {
        return PATH;
    }

    @RequestMapping(value = PATH)
    public ModelAndView error(HttpServletRequest request, HttpServletResponse response) throws ServletException, IOException {

        final List<String> acceptValues = Collections.list(request.getHeaders(HttpHeaders.ACCEPT))
                        .stream()
                        .flatMap(s -> Arrays.stream(s.split(",")))
                        .map(s -> s.trim())
                        .collect(Collectors.toList());

        if (acceptValues.contains(MediaType.APPLICATION_JSON_UTF8_VALUE) || acceptValues.contains(MediaType.APPLICATION_JSON_VALUE)) {
            response.setContentType(MediaType.APPLICATION_JSON_UTF8_VALUE);
            Map<String, Object> errorAttributes = getErrorAttributes(request, debug);
            try {
                if (debug) {
                    objectMapper.writeValue(response.getWriter(), new BlogErrorWithDebug(
                            response.getStatus(),
                            (String) errorAttributes.get("error"),
                            (String) errorAttributes.get("message"),
                            errorAttributes.get("timestamp").toString(),
                            (String) errorAttributes.get("exception"),
                            (String) errorAttributes.get("trace"))
                    );
                } else {
                    objectMapper.writeValue(response.getWriter(), new BlogError(
                            response.getStatus(),
                            (String) errorAttributes.get("error"),
                            (String) errorAttributes.get("message"),
                            errorAttributes.get("timestamp").toString()
                    ));
                }
            } catch (IOException e){
                LOGGER.error("IOException", e);
            }
            return null;

        } else {

            HttpStatus status = getStatus(request);
            if (status.equals(HttpStatus.NOT_FOUND)) {
                // this is not found fallback which works when Accept text/html
                // NotFoundFallback for History API routing, e. g. for url http://127.0.0.1:8080/user/3
                response.setStatus(HttpServletResponse.SC_OK);
                response.setContentType(MediaType.TEXT_HTML_VALUE);
                return new ModelAndView(Constants.Urls.ROOT);
            } else if (status.equals(HttpStatus.FORBIDDEN)) {
                response.setStatus(status.value());
                response.setContentType(MediaType.TEXT_HTML_VALUE);
                response.sendRedirect("/forbidden");
                return new ModelAndView(Constants.Urls.ROOT);
            } else if (status.equals(HttpStatus.UNAUTHORIZED)) {
                response.setStatus(status.value());
                response.setContentType(MediaType.TEXT_HTML_VALUE);
                response.sendRedirect("/unauthorized");
                return new ModelAndView(Constants.Urls.ROOT);
            }

            Map<String, Object> model = Collections.unmodifiableMap(getErrorAttributes(request, debug));
            response.setStatus(status.value());
            ModelAndView modelAndView = resolveErrorView(request, response, status, model);
            return (modelAndView == null ? new ModelAndView("error", model) : modelAndView);
        }
    }

}
