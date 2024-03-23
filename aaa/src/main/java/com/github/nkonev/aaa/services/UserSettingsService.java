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
        if (!Arrays.stream(smileys).allMatch(this::checkAllPossibleRanges)) {
            throw new IllegalArgumentException("Wrong symbol");
        }

        userSettingsRepository.updateSmileys(userId, smileys);
        return userSettingsRepository.findById(userId).get().smileys();
    }

    private boolean checkAllPossibleRanges(String str) {
        return
                    checkSymbol(str, 0x1F600, 0x1F64F) ||
                    checkSymbol(str, 0x1F980, 0x1F9E0) ||
                    checkSymbol(str, 0x1F910, 0x1F96B) ||

                    checkSymbol(str, 0x23E9, 0x23F3) ||
                    checkSymbol(str, 0x23F8, 0x23FA) ||
                    checkSymbol(str, 0x25FB, 0x25FE) ||
                    checkSymbol(str, 0x1F100, 0x1F64F) ||
                    checkSymbol(str, 0x1F680, 0x1F6FF) ||
                    checkSymbol(str, 0x2600,  0x27EF) ||
                    checkSymbol(str, 0x2B00, 0x2BFF) ||
                    "💡".equals(str) ||
                    "☎️".equals(str) ||
                    "🧲".equals(str) ||
                    "#️⃣".equals(str) ||
                    "*️⃣".equals(str) ||
                    "0️⃣".equals(str) ||
                    "1️⃣".equals(str) ||
                    "2️⃣".equals(str) ||
                    "3️⃣".equals(str) ||
                    "4️⃣".equals(str) ||
                    "5️⃣".equals(str) ||
                    "6️⃣".equals(str) ||
                    "7️⃣".equals(str) ||
                    "8️⃣".equals(str) ||
                    "9️⃣".equals(str) ||
                    "🔟".equals(str) ||
                    new String(Character.toChars(0x231A)).equals(str) ||
                    new String(Character.toChars(0x231B)).equals(str) ||
                    new String(Character.toChars(0x00A9)).equals(str) ||
                    new String(Character.toChars(0x00AE)).equals(str) ||
                    new String(Character.toChars(0x2122)).equals(str) ;
    }

    private boolean checkSymbol(String str, int urangeLow, int urangeHigh) {
        for (int iLetter = 0; iLetter < str.length() ; iLetter++) {
            int cp = str.codePointAt(iLetter);
            if (cp >= urangeLow && cp <= urangeHigh) {
                return false;
            }
        }
        return true;
    }

    @Transactional
    public Language setLanguage(long userId, Language language) {
        userSettingsRepository.updateLanguage(userId, language);
        return userSettingsRepository.findById(userId).get().language();
    }
}
