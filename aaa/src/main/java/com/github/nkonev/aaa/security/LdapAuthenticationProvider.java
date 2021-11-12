package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.dao.EmptyResultDataAccessException;
import org.springframework.dao.IncorrectResultSizeDataAccessException;
import org.springframework.ldap.NamingException;
import org.springframework.ldap.core.LdapOperations;
import org.springframework.ldap.query.LdapQuery;
import org.springframework.ldap.query.LdapQueryBuilder;
import org.springframework.security.authentication.AuthenticationProvider;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

import java.util.Locale;
import java.util.Optional;

@Transactional
@Component
public class LdapAuthenticationProvider implements AuthenticationProvider {

    @Autowired
    private LdapOperations ldapOperations;

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Value("${custom.ldap.auth.base:}")
    private String base;

    @Value("${custom.ldap.auth.filter:}")
    private String filter;

    @Value("${custom.ldap.auth.password-encoding.type:}")
    private String passwordEncodingType;

    @Value("${custom.ldap.auth.password-encoding.strength:10}")
    private int passwordEncodingStrength;

    @Value("${custom.ldap.auth.enabled:false}")
    private boolean enabled;

    private static final Logger LOGGER = LoggerFactory.getLogger(LdapAuthenticationProvider.class);

    @Override
    public Authentication authenticate(Authentication authentication) throws AuthenticationException {
        if (enabled) {
            UsernamePasswordAuthenticationToken usernamePasswordAuthenticationToken = (UsernamePasswordAuthenticationToken) authentication;
            var user = usernamePasswordAuthenticationToken.getPrincipal().toString();
            var password = usernamePasswordAuthenticationToken.getCredentials().toString();

            var encodedPassword = encodePassword(password);

            try {
                var lq = LdapQueryBuilder.query().base(base).filter(filter, user);
                ldapOperations.authenticate(lq, encodedPassword);
                UserAccount byUsername = userAccountRepository
                        .findByUsername(user)
                        .orElseGet(() -> userAccountRepository.save(UserAccountConverter.buildUserAccountEntityForLdapInsert(user)));
                UserAccountDetailsDTO userDetails = Optional.of(byUsername)
                        .map(UserAccountConverter::convertToUserAccountDetailsDTO)
                        .orElseThrow(() -> new UsernameNotFoundException("User with login '" + user + "' not found"));
                return new AaaAuthenticationToken(userDetails);
            } catch (IncorrectResultSizeDataAccessException | NamingException e) {
                LOGGER.debug("Unable to authenticate via LDAP", e);
            }
        }
        return null;
    }

    private String encodePassword(String password) {
        switch (passwordEncodingType.toLowerCase()){
            case "bcrypt":
                return new BCryptPasswordEncoder(passwordEncodingStrength).encode(password);
        }
        return password;
    }

    @Override
    public boolean supports(Class<?> authentication) {
        return (UsernamePasswordAuthenticationToken.class.isAssignableFrom(authentication));
    }
}
