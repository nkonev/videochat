package name.nkonev.aaa.services;

import name.nkonev.aaa.exception.UserAlreadyPresentException;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service
public class CheckService {
    private static final Logger LOGGER = LoggerFactory.getLogger(CheckService.class);

    @Autowired
    private UserAccountRepository userAccountRepository;

    public void checkLoginIsFree(String newLogin) {
        if(userAccountRepository.findByUsername(newLogin).isPresent()){
            throw new UserAlreadyPresentException("User with login '" + newLogin + "' is already present");
        }
    }

    public boolean checkEmailIsFree(String email) {
        if (userAccountRepository.findByEmail(email).isPresent()) {
            LOGGER.warn("user with email '{}' already present. exiting...", email);
            return false;
        } else {
            return true;
        }
    }

}
