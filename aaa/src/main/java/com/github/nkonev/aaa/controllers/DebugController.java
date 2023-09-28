package com.github.nkonev.aaa.controllers;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.aaa.dto.OAuth2IdentifiersDTO;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.services.OAuth2ProvidersService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.GetMapping;
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
        var map = objectMapper.readValue(objectMapper.writeValueAsString(myOa), new TypeReference<Map<String, String>>() {}) ;
        map.remove("@class");
        modelAndView.getModelMap().addAttribute("myOauth2Identifiers", map);
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

}
