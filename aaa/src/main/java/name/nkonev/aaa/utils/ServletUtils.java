package name.nkonev.aaa.utils;

import jakarta.servlet.http.HttpServletRequest;
import org.springframework.http.HttpHeaders;
import org.springframework.util.StringUtils;
import org.springframework.web.context.request.RequestAttributes;
import org.springframework.web.context.request.RequestContextHolder;
import org.springframework.web.context.request.ServletRequestAttributes;

import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.stream.Collectors;

public class ServletUtils {


    public static HttpServletRequest getCurrentHttpRequest(){
        RequestAttributes requestAttributes = RequestContextHolder.getRequestAttributes();
        if (requestAttributes instanceof ServletRequestAttributes) {
            return ((ServletRequestAttributes)requestAttributes).getRequest();
        }
        return null;
    }


    public static List<String> getAcceptHeaderValues(HttpServletRequest request) {
        final List<String> acceptValues = Collections.list(request.getHeaders(HttpHeaders.ACCEPT))
                .stream()
                .flatMap(s -> Arrays.stream(s.split(",")))
                .map(s -> s.trim())
                .filter(s -> StringUtils.hasLength(s))
                .collect(Collectors.toList());
        return acceptValues;
    }
}
