package name.nkonev.aaa.services;

import jakarta.servlet.http.HttpServletRequest;
import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.utils.UrlUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.util.StringUtils;

import java.util.List;

@Service
public class RefererService {


    @Autowired
    private AaaProperties aaaProperties;

    public String getRefererOrEmpty(HttpServletRequest currentHttpRequest) {
        if (currentHttpRequest != null){
            String referer = currentHttpRequest.getHeader("Referer");
            String validReferer = getRefererOrEmpty(referer);
            if (StringUtils.hasLength(validReferer)){
                return validReferer;
            }
        }
        return "";
    }

    public String getRefererOrEmpty(String referer) {
        if (StringUtils.hasLength(referer) && isValid(referer)){
            return referer;
        } else {
            return "";
        }
    }

    private boolean isValid(String referer) {
        var allowedUrls = List.of("", aaaProperties.frontendUrl());
        return UrlUtils.containsUrl(allowedUrls, referer);
    }

}
