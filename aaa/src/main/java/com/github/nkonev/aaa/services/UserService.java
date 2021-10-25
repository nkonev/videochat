package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.dto.EditUserDTO;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.exception.BadRequestException;
import com.github.nkonev.aaa.exception.UserAlreadyPresentException;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.security.AaaUserDetailsService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.util.StringUtils;

@Service
public class UserService {
    private static final Logger LOGGER = LoggerFactory.getLogger(UserService.class);

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    public void checkLoginIsFree(EditUserDTO userAccountDTO, UserAccount exists) {
        if (!exists.username().equals(userAccountDTO.login()) && userAccountRepository.findByUsername(userAccountDTO.login()).isPresent()) {
            throw new UserAlreadyPresentException("User with login '" + userAccountDTO.login() + "' is already present");
        }
    }

    public void checkLoginIsFree(EditUserDTO userAccountDTO) {
        if(userAccountRepository.findByUsername(userAccountDTO.login()).isPresent()){
            throw new UserAlreadyPresentException("User with login '" + userAccountDTO.login() + "' is already present");
        }
    }

    public boolean checkEmailIsFree(EditUserDTO userAccountDTO, UserAccount exists) {
        if (exists.email() != null && !exists.email().equals(userAccountDTO.email()) && userAccountDTO.email() != null && userAccountRepository.findByEmail(userAccountDTO.email()).isPresent()) {
            LOGGER.error("user with email '{}' already present. exiting...", exists.email());
            return false;
        } else {
            return true;
        }
    }

    public boolean checkEmailIsFree(EditUserDTO userAccountDTO) {
        if(userAccountRepository.findByEmail(userAccountDTO.email()).isPresent()){
            LOGGER.warn("Skipping sent registration email '{}' because this user already present", userAccountDTO.email());
            return false; // we care for user email leak
        } else {
            return true;
        }
    }

    public void checkLoginIsCorrect(EditUserDTO userAccountDTO) {
        if (StringUtils.isEmpty(userAccountDTO.login())) {
            throw new BadRequestException("empty login");
        }
    }

    public long deleteUser(long userId) {
        aaaUserDetailsService.killSessions(userId);
        userAccountRepository.deleteById(userId);

        return userAccountRepository.count();
    }

}
