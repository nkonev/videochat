package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.dto.Language;
import com.github.nkonev.aaa.dto.LanguageDTO;
import com.github.nkonev.aaa.entity.jdbc.UserSettings;
import com.github.nkonev.aaa.repository.jdbc.UserSettingsRepository;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.Arrays;
import java.util.Optional;

import static com.github.nkonev.aaa.Constants.MAX_SMILEYS_LENGTH;

@Service
public class UserSettingsService {

    @Autowired
    private UserSettingsRepository userSettingsRepository;

    @Transactional
    public LanguageDTO initSettings(long userId) {
        Optional<UserSettings> maybeSettings = userSettingsRepository.findById(userId);
        if (maybeSettings.isEmpty()) {
            userSettingsRepository.insertDefault(userId);
            maybeSettings = userSettingsRepository.findById(userId);
        }
        return new LanguageDTO(maybeSettings.get().language());
    }

    @Transactional
    public String[] getSmileys(long userId) {
        Optional<UserSettings> maybeSettings = userSettingsRepository.findById(userId);
        return maybeSettings.get().smileys();
    }

    @Transactional
    public String[] setSmileys(long userId, String[] smileys) {
        if (smileys.length > MAX_SMILEYS_LENGTH) {
            smileys = Arrays.copyOf(smileys, MAX_SMILEYS_LENGTH);
        }
        userSettingsRepository.updateSmileys(userId, smileys);
        return userSettingsRepository.findById(userId).get().smileys();
    }

    @Transactional
    public Language setLanguage(long userId, Language language) {
        userSettingsRepository.updateLanguage(userId, language);
        return userSettingsRepository.findById(userId).get().language();
    }
}
