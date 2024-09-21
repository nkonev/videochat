package name.nkonev.aaa.entity.rest;

public record KeycloakRoleEntity(
        String id,
        String name,
        String description,
        boolean composite,
        boolean clientRole,
        String containerId
) {
}
