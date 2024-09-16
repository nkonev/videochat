package name.nkonev.aaa.tasks;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.dto.ForceKillSessionsReasonType;
import name.nkonev.aaa.dto.UserRole;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.repository.spring.jdbc.UserListViewRepository;
import name.nkonev.aaa.security.AaaUserDetailsService;
import name.nkonev.aaa.security.RoleMapper;
import name.nkonev.aaa.services.EventService;
import name.nkonev.aaa.utils.NullUtils;
import net.javacrumbs.shedlock.spring.annotation.SchedulerLock;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.ldap.core.LdapOperations;
import org.springframework.ldap.filter.EqualsFilter;
import org.springframework.ldap.filter.OrFilter;
import org.springframework.ldap.query.LdapQueryBuilder;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;
import org.springframework.util.StringUtils;

import java.util.*;
import java.util.stream.Collectors;
import javax.naming.NamingEnumeration;
import javax.naming.NamingException;
import javax.naming.directory.Attributes;

import static name.nkonev.aaa.utils.ConvertUtils.convertToBoolean;
import static name.nkonev.aaa.utils.ConvertUtils.convertToStrings;

@Service
public class SyncLdapTask {
    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private LdapOperations ldapOperations;

    @Autowired
    private UserListViewRepository userListViewRepository;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    private static final Logger LOGGER = LoggerFactory.getLogger(SyncLdapTask.class);
    @Autowired
    private UserAccountRepository userAccountRepository;

    @Scheduled(cron = "${custom.schedulers.sync-ldap.cron}")
    @SchedulerLock(name = "syncLdapTask")
    public void scheduledTask() {
        if (!aaaProperties.schedulers().syncLdap().enabled()) {
            return;
        }

        this.doWork();
    }

    public void doWork() {
        final int pageSize = aaaProperties.schedulers().syncLdap().batchSize();
        LOGGER.debug("Sync ldap task start, batchSize={}", pageSize);

        var shouldContinue = true;
        for (int i = 0; shouldContinue; i++) {
            var chunk = userListViewRepository.findPageWithLdapId(pageSize, i * pageSize);
            shouldContinue = chunk.size() == pageSize;

            var filter = new OrFilter();
            chunk.forEach(userAccount -> {
                filter.or(new EqualsFilter(aaaProperties.ldap().attributeNames().id(), userAccount.ldapId()));
            });

            var lq = LdapQueryBuilder.query().base(aaaProperties.ldap().auth().base()).filter(filter);

            try (var stream = ldapOperations.searchForStream(lq, (Attributes attributes) -> attributes)) {
                var ldapEntries = stream.toList();
                for (var ldapEntry : ldapEntries) {
                    try {
                        var ldapUserId = NullUtils.getOrNullWrapException(() -> ldapEntry.get(aaaProperties.ldap().attributeNames().id()).get().toString());
                        if (StringUtils.hasLength(ldapUserId)) {
                            var o = chunk.stream().filter(ua -> ua.ldapId().equals(ldapUserId)).findAny();
                            if (o.isPresent()) {
                                var userAccount = o.get();
                                var shouldSave = false;

                                if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().username())) {
                                    var ldapUsername = NullUtils.getOrNullWrapException(() -> ldapEntry.get(aaaProperties.ldap().attributeNames().username()).get().toString());
                                    if (StringUtils.hasLength(ldapUsername)) {
                                        if (!ldapUsername.equals(userAccount.username())) {
                                            LOGGER.info("For userId={}, ldapId={}, setting username={}", userAccount.id(), ldapUserId, ldapUsername);
                                            userAccount = userAccount.withUsername(ldapUsername);
                                            shouldSave = true;
                                        }
                                    } else {
                                        LOGGER.warn("For userId={}, ldapId={}, got empty ldap username", userAccount.id(), ldapUserId);
                                    }
                                }

                                if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().email())) {
                                    var ldapEmail = NullUtils.getOrNullWrapException(() -> ldapEntry.get(aaaProperties.ldap().attributeNames().email()).get().toString());
                                    if (!Objects.equals(ldapEmail, userAccount.email())) {
                                        LOGGER.info("For userId={}, ldapId={}, setting email={}", userAccount.id(), ldapUserId, ldapEmail);
                                        userAccount = userAccount.withEmail(ldapEmail);
                                        shouldSave = true;
                                    }
                                }

                                if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().role())) {
                                    Set<String> rawRoles = new HashSet<>();
                                    var groups = ldapEntry.get(aaaProperties.ldap().attributeNames().role()).getAll();
                                    if (groups != null) {
                                        rawRoles.addAll(convertToStrings(groups));
                                    }
                                    var mappedRoles = RoleMapper.map(aaaProperties.roleMappings().ldap(), rawRoles);
                                    var oldRoles = Arrays.stream(userAccount.roles()).collect(Collectors.toSet());
                                    if (!oldRoles.equals(mappedRoles)) {
                                        LOGGER.info("For userId={}, ldapId={}, setting roles={}", userAccount.id(), ldapUserId, mappedRoles);
                                        aaaUserDetailsService.killSessions(userAccount.id(), ForceKillSessionsReasonType.user_roles_changed);
                                        userAccount = userAccount.withRoles(mappedRoles.toArray(UserRole[]::new));
                                        shouldSave = true;
                                    }
                                }

                                if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().locked())) {
                                    var ldapLockedV = NullUtils.getOrNullWrapException(() -> ldapEntry.get(aaaProperties.ldap().attributeNames().locked()).get().toString());
                                    boolean ldapLocked = convertToBoolean(ldapLockedV);
                                    if (ldapLocked != userAccount.locked()) {
                                        LOGGER.info("For userId={}, ldapId={}, setting locked={}", userAccount.id(), ldapUserId, ldapLocked);
                                        if (ldapLocked) {
                                            aaaUserDetailsService.killSessions(userAccount.id(), ForceKillSessionsReasonType.user_locked);
                                        }
                                        userAccount = userAccount.withLocked(ldapLocked);
                                        shouldSave = true;
                                    }
                                }

                                if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().enabled())) {
                                    var ldapEnabledV = NullUtils.getOrNullWrapException(() -> ldapEntry.get(aaaProperties.ldap().attributeNames().enabled()).get().toString());
                                    boolean ldapEnabled = convertToBoolean(ldapEnabledV);
                                    if (ldapEnabled != userAccount.enabled()) {
                                        LOGGER.info("For userId={}, ldapId={}, setting enabled={}", userAccount.id(), ldapUserId, ldapEnabled);
                                        if (!ldapEnabled) {
                                            aaaUserDetailsService.killSessions(userAccount.id(), ForceKillSessionsReasonType.user_locked);
                                        }
                                        userAccount = userAccount.withEnabled(ldapEnabled);
                                        shouldSave = true;
                                    }
                                }

                                if (shouldSave) {
                                    LOGGER.info("Saving userId={}, ldapId={}", userAccount.id(), ldapUserId);
                                    userAccountRepository.save(userAccount);
                                }
                            } else {
                                LOGGER.warn("Unable to find the corresponding userAccount for ldapId = {}", ldapUserId);
                            }
                        } else {
                            LOGGER.warn("Got empty ldap userId");
                        }
                    } catch (Exception e) {
                        LOGGER.error(e.getMessage(), e);
                    }
                }
            }
        }

        LOGGER.debug("Sync ldap task finish");
    }

}

