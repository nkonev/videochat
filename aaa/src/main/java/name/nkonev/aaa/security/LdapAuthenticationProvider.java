package name.nkonev.aaa.security;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.exception.UserAlreadyPresentException;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.services.EventService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.dao.IncorrectResultSizeDataAccessException;
import org.springframework.ldap.NamingException;
import org.springframework.ldap.core.LdapOperations;
import org.springframework.ldap.query.LdapQueryBuilder;
import org.springframework.security.authentication.AuthenticationProvider;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.stereotype.Component;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.util.StringUtils;

import java.util.Arrays;
import java.util.HashSet;
import java.util.Set;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.stream.Collectors;

// https://spring.io/guides/gs/authenticating-ldap
@Component
public class LdapAuthenticationProvider implements AuthenticationProvider {

    @Autowired
    private LdapOperations ldapOperations;

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private EventService eventService;

    @Autowired
    private TransactionTemplate transactionTemplate;

    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private UserAccountConverter userAccountConverter;

    private static final Logger LOGGER = LoggerFactory.getLogger(LdapAuthenticationProvider.class);

    @Override
    public Authentication authenticate(Authentication authentication) throws AuthenticationException {
        if (aaaProperties.ldap().auth().enabled()) {
            UsernamePasswordAuthenticationToken usernamePasswordAuthenticationToken = (UsernamePasswordAuthenticationToken) authentication;
            var userName = usernamePasswordAuthenticationToken.getPrincipal().toString();
            var password = usernamePasswordAuthenticationToken.getCredentials().toString();

            var encodedPassword = encodePassword(password);

            AtomicBoolean created = new AtomicBoolean();

            try {
                var userAccount = transactionTemplate.execute(status -> {
                    var lq = LdapQueryBuilder.query().base(aaaProperties.ldap().auth().base()).filter(aaaProperties.ldap().auth().filter(), userName);
                    ldapOperations.authenticate(lq, encodedPassword);
                    var ctx = ldapOperations.searchForContext(lq);
                    var ldapUserId = ctx.getObjectAttribute(aaaProperties.ldap().attributeNames().id()).toString();

                    final Set<String> rawRoles = new HashSet<>();
                    if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().role())) {
                        String[] groups = ctx.getStringAttributes(aaaProperties.ldap().attributeNames().role());
                        if (groups != null) {
                            rawRoles.addAll(Arrays.stream(groups).collect(Collectors.toSet()));
                        }
                    }

                    UserAccount byLdapId = userAccountRepository
                        .findByLdapId(ldapUserId)
                        .orElseGet(() -> {
                            // create a new
                            userAccountRepository.findByUsername(userName).ifPresent(ua -> {
                                throw new UserAlreadyPresentException("User with login '" + userName + "' is already present");
                            });
                            var mappedRoles = RoleMapper.map(aaaProperties.roleMappings().ldap(), rawRoles);
                            var user = userAccountRepository.save(UserAccountConverter.buildUserAccountEntityForLdapInsert(userName, ldapUserId, mappedRoles));
                            created.set(true);
                            return user;
                        });
                    return byLdapId;
                });
                UserAccountDetailsDTO userDetails = userAccountConverter.convertToUserAccountDetailsDTO(userAccount);

                if (created.get()) {
                    eventService.notifyProfileCreated(userAccount);
                }

                return new AaaAuthenticationToken(userDetails);
            } catch (UserAlreadyPresentException e) {
                LOGGER.warn("User already exists: {}", e.getMessage());
            } catch (IncorrectResultSizeDataAccessException | NamingException e) {
                LOGGER.debug("Unable to authenticate via LDAP", e);
            }
        }
        return null;
    }

    private String encodePassword(String password) {
        switch (aaaProperties.ldap().password().encodingType().toLowerCase()){
            case "bcrypt":
                return new BCryptPasswordEncoder(aaaProperties.ldap().password().strength()).encode(password);
            default:
                return password;
        }
    }

    @Override
    public boolean supports(Class<?> authentication) {
        return (UsernamePasswordAuthenticationToken.class.isAssignableFrom(authentication));
    }
}
