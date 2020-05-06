package com.github.nkonev.blog.util;

import org.springframework.boot.web.server.Ssl;
import org.springframework.boot.web.servlet.server.AbstractServletWebServerFactory;

public class ContextPathHelper {
    public static String urlWithContextPath(AbstractServletWebServerFactory abstractConfigurableEmbeddedServletContainer){
        Ssl ssl = abstractConfigurableEmbeddedServletContainer.getSsl();
        String protocol = ssl!=null && ssl.isEnabled() ? "https" : "http";
        return protocol+"://127.0.0.1:"+abstractConfigurableEmbeddedServletContainer.getPort()+abstractConfigurableEmbeddedServletContainer.getContextPath();
    }

}
