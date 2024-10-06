package name.nkonev.aaa.entity.rest;

import name.nkonev.aaa.dto.ExternalSyncEntity;

public record KeycloakUserInRoleEntity(
        String id,
        String username,
        String firstName,
        String lastName,
        String email,
        Boolean emailVerified,
        Boolean enabled
) implements ExternalSyncEntity {
    @Override
    public String getId() {
        return id;
    }
}
