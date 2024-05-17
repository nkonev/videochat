package name.nkonev.aaa.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "custom.ldap.auth")
public record LdapAuthProperties(
    String base,
    boolean enabled,
    String filter,
    String uidName
) { }
