package com.github.nkonev.blog.utils;

import org.springframework.web.context.request.RequestAttributes;
import org.springframework.web.context.request.RequestContextHolder;
import org.springframework.web.context.request.ServletRequestAttributes;

import javax.servlet.http.HttpServletRequest;

public class ServletUtils {
    public static String nullToEmpty(String s){
        if (s == null) {
            return "";
        } else {
            return s;
        }
    }

    public static String getQuery(HttpServletRequest request) {
        if (request.getQueryString() == null) {
            return "";
        } else {
            return "?" + request.getQueryString();
        }
    }

    public static String getPath(HttpServletRequest request) {
        String s = request.getRequestURI();
        if (s == null){
            return "";
        }
        if (s.endsWith("/")){
            return s.substring(0, s.length()-1);
        }
        return s;
    }


    public static HttpServletRequest getCurrentHttpRequest(){
        RequestAttributes requestAttributes = RequestContextHolder.getRequestAttributes();
        if (requestAttributes instanceof ServletRequestAttributes) {
            return ((ServletRequestAttributes)requestAttributes).getRequest();
        }
        return null;
    }

    public static String getIpAddressFromRequestContext() {
        if (RequestContextHolder.getRequestAttributes() == null) {return "cannot-get-remote-addr";}
        HttpServletRequest request = ((ServletRequestAttributes) RequestContextHolder.getRequestAttributes()).getRequest();
        return getIpAddress(request);
    }


    // http://www.mkyong.com/java/how-to-get-client-ip-address-in-java/
    // https://stackoverflow.com/a/29910902
    public static String getIpAddress(HttpServletRequest request) {
        String remoteAddr = "cannot-get-remote-addr";

        if (request != null) {
            remoteAddr = request.getHeader("X-FORWARDED-FOR");
            if (remoteAddr == null || "".equals(remoteAddr)) {
                remoteAddr = request.getRemoteAddr();
            }
        }

        return remoteAddr;
    }
}
