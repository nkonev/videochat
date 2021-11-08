ALTER TABLE users ADD COLUMN keycloak_id VARCHAR(64);
ALTER TYPE user_creation_type ADD VALUE 'KEYCLOAK';
ALTER TABLE users ADD UNIQUE (keycloak_id) ;