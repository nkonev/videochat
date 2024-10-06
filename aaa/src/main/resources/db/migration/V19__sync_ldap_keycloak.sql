alter table user_account
    add column sync_ldap_date_time timestamp without time zone,
    add column sync_ldap_roles_date_time timestamp without time zone,
    add column sync_keycloak_date_time timestamp without time zone,
    add column sync_keycloak_roles_date_time timestamp without time zone;
