package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.UserSettings;
import com.github.nkonev.aaa.repository.jdbc.UserSettingsRepository;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

import java.util.Arrays;
import java.util.Optional;

import static com.github.nkonev.aaa.Constants.MAX_SMILEYS_LENGTH;

@RestController
@Transactional
public class UserSettingsController {

    @Autowired
    private UserSettingsRepository userSettingsRepository;

    @PreAuthorize("isAuthenticated()")
    @GetMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.SETTINGS + Constants.Urls.SMILEYS, produces = MediaType.APPLICATION_JSON_VALUE)
    public String[] getSmileys(@AuthenticationPrincipal UserAccountDetailsDTO userAccount) {
        Optional<UserSettings> maybeSettings = userSettingsRepository.findById(userAccount.getId());
        if (maybeSettings.isEmpty()) {
            userSettingsRepository.insertDefault(userAccount.getId());
            maybeSettings = userSettingsRepository.findById(userAccount.getId());
        }
        return maybeSettings.get().smileys();
    }

    @PreAuthorize("isAuthenticated()")
    @PutMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.SETTINGS + Constants.Urls.SMILEYS, produces = MediaType.APPLICATION_JSON_VALUE, consumes = MediaType.APPLICATION_JSON_VALUE)
    public String[] setSmileys(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestBody String[] smileys) {
        if (smileys.length > MAX_SMILEYS_LENGTH) {
            smileys = Arrays.copyOf(smileys, MAX_SMILEYS_LENGTH);
        }
        userSettingsRepository.save(new UserSettings(userAccount.getId(), smileys));
        return userSettingsRepository.findById(userAccount.getId()).get().smileys();
    }
}
