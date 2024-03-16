package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.dto.EditUserDTO;
import com.github.nkonev.aaa.dto.Language;
import com.github.nkonev.aaa.services.RegistrationService;
import jakarta.servlet.http.HttpSession;
import jakarta.validation.Valid;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@Controller
public class RegistrationController {

    @Autowired
    private RegistrationService registrationService;

    @PostMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.REGISTER)
    @ResponseBody
    public void register(@RequestBody @Valid EditUserDTO editUserDTO, @RequestParam(defaultValue = Language.DEFAULT) Language language) {
        registrationService.register(editUserDTO, language);
    }

    @GetMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.REGISTER_CONFIRM)
    public String confirm(@RequestParam(Constants.Urls.UUID) UUID uuid, HttpSession httpSession) {
        return "redirect:" + registrationService.confirm(uuid, httpSession);
    }

    @PostMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.RESEND_CONFIRMATION_EMAIL)
    @ResponseBody
    public void resendConfirmationToken(@RequestParam String email, @RequestParam(defaultValue = Language.DEFAULT) Language language) {
        registrationService.resendConfirmationToken(email, language);
    }
}
