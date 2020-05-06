-- add casts for eliminate spring-data-jdbc's errors like
--  Caused by: org.postgresql.util.PSQLException: ERROR: column "creation_type" is of type auth.user_creation_type but expression is of type character varying
CREATE CAST (character varying AS auth.user_creation_type) WITH INOUT AS ASSIGNMENT;
CREATE CAST (character varying AS auth.user_role) WITH INOUT AS ASSIGNMENT;