package name.nkonev.aaa.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "custom.ldap.auth.password-encoding")
public record LdapPasswordEncodingProperties(
    String type,
    int strength
) { }
