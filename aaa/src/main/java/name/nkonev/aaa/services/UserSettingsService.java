package name.nkonev.aaa.services;

import name.nkonev.aaa.dto.Language;
import name.nkonev.aaa.dto.SettingsDTO;
import name.nkonev.aaa.entity.jdbc.UserSettings;
import name.nkonev.aaa.repository.jdbc.UserSettingsRepository;
import org.owasp.encoder.Encode;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.Arrays;
import java.util.Optional;

import static name.nkonev.aaa.Constants.MAX_SMILEYS_LENGTH;
import static name.nkonev.aaa.Constants.USERS_ONLINE_LENGTH;

@Service
public class UserSettingsService {

    private static final Logger LOGGER = LoggerFactory.getLogger(UserSettingsService.class);

    @Autowired
    private UserSettingsRepository userSettingsRepository;

    // despite creating settings with language in RegistrationService.register() this method still exists for
    // the case registration via OAuth, Ldap and so on
    @Transactional
    public SettingsDTO initSettings(long userId) {
        Optional<UserSettings> maybeSettings = userSettingsRepository.findById(userId);
        if (maybeSettings.isEmpty()) {
            userSettingsRepository.insertDefault(userId);
            maybeSettings = userSettingsRepository.findById(userId);
        }
        var userSettings = maybeSettings.get();
        return new SettingsDTO(userSettings.language());
    }

    @Transactional
    public String[] getSmileys(long userId) {
        Optional<UserSettings> maybeSettings = userSettingsRepository.findById(userId);
        return maybeSettings.get().smileys();
    }

    @Transactional
    public String[] setSmileys(long userId, String[] smileys) {
        String[] smileysReal;
        if (smileys.length > MAX_SMILEYS_LENGTH) {
            smileysReal = Arrays.copyOf(smileys, MAX_SMILEYS_LENGTH);
            LOGGER.info("Cutting {} userIds to {}", smileys.length, MAX_SMILEYS_LENGTH);
        } else {
             smileysReal = smileys;
        }
        smileysReal = Arrays.stream(smileysReal).map(Encode::forHtml).toList().toArray(new String[0]);

        userSettingsRepository.updateSmileys(userId, smileysReal);
        return userSettingsRepository.findById(userId).get().smileys();
    }

    @Transactional
    public Language setLanguage(long userId, Language language) {
        userSettingsRepository.updateLanguage(userId, language);
        return userSettingsRepository.findById(userId).get().language();
    }

}
