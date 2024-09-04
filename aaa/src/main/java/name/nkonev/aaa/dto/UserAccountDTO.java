package name.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonIgnore;
import name.nkonev.aaa.Constants;
import jakarta.validation.constraints.NotEmpty;

import java.time.LocalDateTime;

/**
 * Created by nik on 22.06.17.
 * Contains public information
 */
public record UserAccountDTO (

    Long id,

    @NotEmpty
    String login,

    String avatar,

    String avatarBig,

    String shortInfo,

    @JsonFormat(shape=JsonFormat.Shape.STRING, pattern= Constants.DATE_FORMAT)
    LocalDateTime lastLoginDateTime,

    OAuth2IdentifiersDTO oauth2Identifiers,
    String loginColor,
    boolean ldap
) {
    public UserAccountDTO(Long id, String login, String avatar, String avatarBig, String shortInfo, LocalDateTime lastLoginDateTime, OAuth2IdentifiersDTO oauth2Identifiers, String loginColor, boolean ldap) {
        this.id = id;
        this.login = login;
        this.avatar = avatar;
        this.avatarBig = avatarBig;
        this.shortInfo = shortInfo;
        this.lastLoginDateTime = lastLoginDateTime;
        this.oauth2Identifiers = oauth2Identifiers;
        this.loginColor = loginColor;
        this.ldap = ldap;
    }

    @JsonIgnore // to use in Freemarker template header.ftlh
    public String getUsername() {
        return login;
    }

    @JsonIgnore // to use in Freemarker template header.ftlh
    public Long getIdentificator() {
        return id;
    }
}
