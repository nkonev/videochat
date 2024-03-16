package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.dto.Language;
import com.github.nkonev.aaa.dto.PasswordResetDTO;
import com.github.nkonev.aaa.services.PasswordResetService;
import jakarta.servlet.http.HttpSession;
import jakarta.validation.Valid;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class PasswordResetController {

    @Autowired
    private PasswordResetService passwordResetService;

    /**
     * https://www.owasp.org/index.php/Forgot_Password_Cheat_Sheet
     * https://stackoverflow.com/questions/1102781/best-way-for-a-forgot-password-implementation/1102817#1102817
     * Yes, if your email is stolen you can lost your account
     * @param email
     */
    @PostMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.REQUEST_PASSWORD_RESET)
    public void requestPasswordReset(@RequestParam String email, @RequestParam(defaultValue = Language.DEFAULT) Language language) {
        passwordResetService.requestPasswordReset(email, language);
    }

    @PostMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.PASSWORD_RESET_SET_NEW)
    public void resetPassword(@RequestBody @Valid PasswordResetDTO passwordResetDto, HttpSession httpSession) {
        passwordResetService.resetPassword(passwordResetDto, httpSession);
    }

}
