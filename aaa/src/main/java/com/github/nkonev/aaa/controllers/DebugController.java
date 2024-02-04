package com.github.nkonev.aaa.controllers;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.dto.OAuth2IdentifiersDTO;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.services.AsyncEmailService;
import com.github.nkonev.aaa.services.OAuth2ProvidersService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.ResponseBody;
import org.springframework.web.servlet.ModelAndView;
import com.github.nkonev.aaa.dto.UserAccountDTO;
import java.util.Map;
import java.util.Optional;

@Controller
public class DebugController {

    @Autowired
    private OAuth2ProvidersService oAuth2ProvidersService;

    @Autowired
    private ObjectMapper objectMapper;

    @Autowired
    private AsyncEmailService asyncEmailService;

    @GetMapping({"/index.html", "/"})
    public ModelAndView ModelAndView(@AuthenticationPrincipal UserAccountDetailsDTO userAccount) throws JsonProcessingException {
        ModelAndView modelAndView = new ModelAndView("index");
        setCommonHeaderData(userAccount, modelAndView);
        return modelAndView;
    }

    private void setCommonHeaderData(UserAccountDetailsDTO userAccount, ModelAndView modelAndView) throws JsonProcessingException {
        var myOa = Optional
            .ofNullable(userAccount)
            .map(UserAccountDetailsDTO::userAccountDTO)
            .map(UserAccountDTO::oauth2Identifiers)
            .orElse(new OAuth2IdentifiersDTO());
        var myOauth2IdentifiersMap = objectMapper.readValue(objectMapper.writeValueAsString(myOa), new TypeReference<Map<String, String>>() {}) ;
        myOauth2IdentifiersMap.remove("@class");

        var myPr = Optional
            .ofNullable(userAccount)
            .map(UserAccountDetailsDTO::userAccountDTO)
            .orElse(null);

        modelAndView.getModelMap().addAttribute("myOauth2Identifiers", myOauth2IdentifiersMap);
        modelAndView.getModelMap().addAttribute("myPrincipal", myPr);
    }

    @GetMapping({"/oauth2.html"})
    public ModelAndView oauth2(@AuthenticationPrincipal UserAccountDetailsDTO userAccount) throws JsonProcessingException {
        ModelAndView modelAndView = new ModelAndView("oauth2");
        setCommonHeaderData(userAccount, modelAndView);

        var providers = oAuth2ProvidersService.availableOauth2Providers();
        modelAndView.getModelMap().addAttribute("availableOauth2Providers", providers);

        return modelAndView;
    }

    @GetMapping({"/login.html"})
    public ModelAndView login(@AuthenticationPrincipal UserAccountDetailsDTO userAccount) throws JsonProcessingException {
        ModelAndView modelAndView = new ModelAndView("login");
        setCommonHeaderData(userAccount, modelAndView);
        return modelAndView;
    }

    @ResponseBody
    @PutMapping(value = Constants.Urls.INTERNAL_API + "/email", consumes = MediaType.APPLICATION_JSON_VALUE, produces = MediaType.APPLICATION_JSON_VALUE)
    public void checkEmail(@RequestBody AsyncEmailService.ArbitraryEmailDto emailDto) {
        asyncEmailService.sendArbitraryEmail(emailDto);
    }

}
