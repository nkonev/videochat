package com.github.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.github.nkonev.aaa.Constants;
import javax.validation.constraints.NotEmpty;
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

    @JsonFormat(shape=JsonFormat.Shape.STRING, pattern= Constants.DATE_FORMAT)
    LocalDateTime lastLoginDateTime,

    OAuth2IdentifiersDTO oauth2Identifiers
) {
    public UserAccountDTO(Long id, String login, String avatar, String avatarBig, LocalDateTime lastLoginDateTime, OAuth2IdentifiersDTO oauth2Identifiers) {
        this.id = id;
        this.login = login;
        this.avatar = avatar;
        this.avatarBig = avatarBig;
        this.lastLoginDateTime = lastLoginDateTime;
        this.oauth2Identifiers = oauth2Identifiers;
    }

}
