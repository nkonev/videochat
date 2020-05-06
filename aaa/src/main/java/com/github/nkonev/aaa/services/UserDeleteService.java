package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.security.BlogUserDetailsService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

@Component
public class UserDeleteService {
    @Autowired
    private BlogUserDetailsService blogUserDetailsService;

    @Autowired
    private UserAccountRepository userAccountRepository;

    public long deleteUser(long userId) {
        blogUserDetailsService.killSessions(userId);
        UserAccount deleted = userAccountRepository.findByUsername(Constants.DELETED).orElseThrow();
//        postRepository.moveToAnotherUser(userId, deleted.getId());
//        commentRepository.moveToAnotherUser(userId, deleted.getId());
        userAccountRepository.deleteById(userId);

        return userAccountRepository.count();
    }

}
