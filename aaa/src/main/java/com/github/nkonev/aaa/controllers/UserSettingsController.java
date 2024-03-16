package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.services.UserSettingsService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class UserSettingsController {

    @Autowired
    private UserSettingsService userSettingsService;

    @PreAuthorize("isAuthenticated()")
    @GetMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.SETTINGS + Constants.Urls.SMILEYS, produces = MediaType.APPLICATION_JSON_VALUE)
    public String[] getSmileys(@AuthenticationPrincipal UserAccountDetailsDTO userAccount) {
        return userSettingsService.getSmileys(userAccount.getId());
    }

    @PreAuthorize("isAuthenticated()")
    @PutMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.SETTINGS + Constants.Urls.SMILEYS, produces = MediaType.APPLICATION_JSON_VALUE, consumes = MediaType.APPLICATION_JSON_VALUE)
    public String[] setSmileys(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestBody String[] smileys) {
        return userSettingsService.setSmileys(userAccount.getId(), smileys);
    }
}
