package name.nkonev.aaa.services;

import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.exception.UserAlreadyPresentException;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.*;

@Service
public class CheckService {
    private static final Logger LOGGER = LoggerFactory.getLogger(CheckService.class);

    @Autowired
    private UserAccountRepository userAccountRepository;

    public boolean checkLoginIsFree(String login) {
        return checkLogin(login).isEmpty();
    }

    public Optional<UserAccount> checkLogin(String login) {
        var t = userAccountRepository.findByUsername(login);
        if (t.isPresent()) {
            LOGGER.info("user with login '{}' already present", login);
        }
        return t;
    }

    public Map<String, UserAccount> checkLogins(List<String> logins) {
        var res = new HashMap<String, UserAccount>();
        if (logins.isEmpty()) {
            return res;
        }
        List<String> notNullLogins = logins.stream().filter(Objects::nonNull).toList();
        var users = userAccountRepository.findByUsernameInOrderById(notNullLogins);
        for (var u : users) {
            res.put(u.username(), u);
        }
        return res;
    }

    public void checkLoginIsFreeOrThrow(String login) {
        if (!checkLoginIsFree(login)){
            throw new UserAlreadyPresentException("User with login '" + login + "' is already present");
        }
    }

    public boolean checkEmailIsFree(String email) {
        return checkEmail(email).isEmpty();
    }

    public Optional<UserAccount> checkEmail(String email) {
        var t = userAccountRepository.findByEmail(email);
        if (t.isPresent()) {
            LOGGER.info("user with email '{}' already present", email);
        }
        return t;
    }

    public Map<String, UserAccount> checkEmails(List<String> emails) {
        var res = new HashMap<String, UserAccount>();
        if (emails.isEmpty()) {
            return res;
        }
        List<String> notNullEmails = emails.stream().filter(Objects::nonNull).toList();
        var users = userAccountRepository.findByEmailInOrderById(notNullEmails);
        for (var u : users) {
            res.put(u.email(), u);
        }
        return res;
    }

    public void checkEmailIsFreeOrThrow(String email) {
        if (!checkEmailIsFree(email)) {
            throw new UserAlreadyPresentException("User with email '" + email + "' is already present");
        }
    }
}
