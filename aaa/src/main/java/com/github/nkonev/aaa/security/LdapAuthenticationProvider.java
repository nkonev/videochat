package com.github.nkonev.aaa.security;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
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
import org.springframework.stereotype.Component;

@Component
public class LdapAuthenticationProvider implements AuthenticationProvider {

    @Autowired
    private LdapOperations ldapOperations;

    private static final Logger LOGGER = LoggerFactory.getLogger(LdapAuthenticationProvider.class);

    @Override
    public Authentication authenticate(Authentication authentication) throws AuthenticationException {
        // TODO search in LDAP if it's enabled
        //  if success - create in db, then return it
        //  otherwise return null

        UsernamePasswordAuthenticationToken usernamePasswordAuthenticationToken = (UsernamePasswordAuthenticationToken) authentication;
        var user = usernamePasswordAuthenticationToken.getPrincipal().toString();
        var password = usernamePasswordAuthenticationToken.getCredentials().toString();

        try {
            var lq = LdapQueryBuilder.query().base("").filter("uid={0}", user);
            ldapOperations.authenticate(lq, password);

        } catch (IncorrectResultSizeDataAccessException | NamingException e) {
            LOGGER.debug("Unable to authenticate via LDAP", e);
        }
        return null;
    }

    @Override
    public boolean supports(Class<?> authentication) {
        return (UsernamePasswordAuthenticationToken.class.isAssignableFrom(authentication));
    }
}
