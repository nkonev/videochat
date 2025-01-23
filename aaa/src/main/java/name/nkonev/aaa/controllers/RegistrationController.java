package name.nkonev.aaa.controllers;

import jakarta.servlet.http.HttpServletRequest;
import name.nkonev.aaa.Constants;
import name.nkonev.aaa.dto.EditUserDTO;
import name.nkonev.aaa.dto.Language;
import name.nkonev.aaa.services.RegistrationService;
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

    @PostMapping(value = Constants.Urls.EXTERNAL_API + Constants.Urls.REGISTER)
    @ResponseBody
    public void register(@RequestBody @Valid EditUserDTO editUserDTO, @RequestParam(defaultValue = Language.DEFAULT) Language language, @RequestParam(required = false) String referer, HttpServletRequest httpServletRequest) {
        registrationService.register(editUserDTO, language, referer, httpServletRequest);
    }

    @GetMapping(value = Constants.Urls.EXTERNAL_API + Constants.Urls.REGISTER_CONFIRM)
    public String confirm(@RequestParam(Constants.Urls.UUID) UUID uuid, HttpSession httpSession, HttpServletRequest httpServletRequest) {
        return "redirect:" + registrationService.confirm(uuid, httpSession, httpServletRequest);
    }

    @PostMapping(value = Constants.Urls.EXTERNAL_API + Constants.Urls.RESEND_CONFIRMATION_EMAIL)
    @ResponseBody
    public void resendConfirmationToken(@RequestParam String email, @RequestParam(defaultValue = Language.DEFAULT) Language language, HttpServletRequest httpServletRequest) {
        registrationService.resendConfirmationToken(email, language, "", httpServletRequest);
    }
}
