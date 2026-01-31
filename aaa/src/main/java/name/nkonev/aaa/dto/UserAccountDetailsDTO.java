package name.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonAutoDetect;
import name.nkonev.aaa.entity.jdbc.CreationType;
import name.nkonev.aaa.utils.LdapUtils;
import name.nkonev.aaa.utils.PermissionsUtils;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.oauth2.core.oidc.OidcIdToken;
import org.springframework.security.oauth2.core.oidc.OidcUserInfo;
import org.springframework.security.oauth2.core.oidc.user.OidcUser;
import org.springframework.security.oauth2.core.user.OAuth2User;

import java.time.LocalDateTime;
import java.util.Collection;
import java.util.HashMap;
import java.util.Map;

/**
 * Internal class for Spring Security, it shouldn't be passed to browser via Rest API
 */
@JsonAutoDetect(fieldVisibility = JsonAutoDetect.Visibility.ANY, getterVisibility = JsonAutoDetect.Visibility.NONE, setterVisibility = JsonAutoDetect.Visibility.NONE, isGetterVisibility = JsonAutoDetect.Visibility.NONE)
public record UserAccountDetailsDTO (
    UserAccountDTO userAccountDTO,

    // OAuth2 specific Facebook and Vkontakte
    Map<String, Object> oauth2Attributes,
    // OAuth2 specific Google
    OidcIdToken idToken,
    OidcUserInfo userInfo,

    String password, // password hash
    boolean expired,
    boolean locked,
    boolean enabled,
    boolean confirmed,

    Collection<GrantedAuthority> roles,
    String email,
    boolean awaitingForConfirmEmailChange,
    String ldapId,
    CreationType creationType,
    ExternalPermission[] overrideAddPermissions,
    ExternalPermission[] overrideRemovePermissions
) implements UserDetails, OAuth2User, OidcUser {

    public UserAccountDetailsDTO(
            Long id,
            String login,
            String avatar,
            String avatarBig,
            String shortInfo,
            String password,
            boolean expired,
            boolean locked,
            boolean enabled,
            boolean confirmed,
            Collection<GrantedAuthority> roles,
            String email,
            boolean awaitingForConfirmEmailChange,
            LocalDateTime lastSeenDateTime,
            OAuth2IdentifiersDTO oauthIdentifiers,
            String loginColor,
            String ldapId,
            AdditionalDataDTO additionalDataDTO,
            CreationType creationType,
            ExternalPermission[] overrideAddPermissions,
            ExternalPermission[] overrideRemovePermissions
    ) {
        this(
                new UserAccountDTO(
                    id,
                    login,
                    avatar,
                    avatarBig,
                    shortInfo,
                    lastSeenDateTime,
                    oauthIdentifiers,
                    loginColor,
                    LdapUtils.isLdapSet(ldapId),
                    additionalDataDTO,
                    PermissionsUtils.areOverriddenPermissions(overrideAddPermissions, overrideRemovePermissions)
                ),
                new HashMap<>(),
                null,
                null,
                password,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                awaitingForConfirmEmailChange,
                ldapId,
                creationType,
                overrideAddPermissions,
                overrideRemovePermissions
        );
    }

    @Override
    public String getPassword() {
        return password;
    }

    @Override
    public String getUsername() {
        return this.userAccountDTO.login();
    }

    @Override
    public boolean isAccountNonExpired() {
        return !expired;
    }

    @Override
    public boolean isAccountNonLocked() {
        return !locked;
    }

    @Override
    public boolean isCredentialsNonExpired() {
        return true;
    }

    @Override
    public boolean isEnabled() {
        return enabled && confirmed;
    }

    @Override
    public Map<String, Object> getAttributes() {
        return oauth2Attributes;
    }

    @Override
    public Collection<? extends GrantedAuthority> getAuthorities() {
        return roles;
    }

    @Override
    public String getName() {
        return this.userAccountDTO.login();
    }

    public Collection<GrantedAuthority> getRoles() {
        return roles;
    }

    public String getEmail() {
        return email;
    }

    @Override
    public Map<String, Object> getClaims() {
        return oauth2Attributes;
    }

    @Override
    public OidcUserInfo getUserInfo() {
        return userInfo;
    }

    @Override
    public OidcIdToken getIdToken() {
        return idToken;
    }

    public Long getId() {
        return userAccountDTO.id();
    }

    public String getAvatar() {
        return userAccountDTO.avatar();
    }

    public String getAvatarBig() {
        return userAccountDTO.avatarBig();
    }

    public OAuth2IdentifiersDTO getOauth2Identifiers() {
        return userAccountDTO.oauth2Identifiers();
    }

    public LocalDateTime getLastSeenDateTime() {
        return userAccountDTO.lastSeenDateTime();
    }

    public String getLoginColor() {
        return userAccountDTO.loginColor();
    }

    public UserAccountDetailsDTO withOauth2Identifiers(OAuth2IdentifiersDTO newOauth2Identifiers) {
        return new UserAccountDetailsDTO(
                this.userAccountDTO.withOauth2Identifiers(newOauth2Identifiers),
                oauth2Attributes,
                idToken,
                userInfo,
                password,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                awaitingForConfirmEmailChange,
                ldapId,
                creationType,
                overrideAddPermissions,
                overrideRemovePermissions
        );
    }

    public UserAccountDetailsDTO withUserAccountDTO(UserAccountDTO newUserAccountDTO) {
        return new UserAccountDetailsDTO(
                newUserAccountDTO,
                oauth2Attributes,
                idToken,
                userInfo,
                password,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                awaitingForConfirmEmailChange,
                ldapId,
                creationType,
                overrideAddPermissions,
                overrideRemovePermissions
        );
    }

}
