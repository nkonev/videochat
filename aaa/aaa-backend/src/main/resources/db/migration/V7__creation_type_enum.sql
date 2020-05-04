SET search_path = auth, pg_catalog;

CREATE TYPE user_creation_type AS ENUM (
    'REGISTRATION',
    'FACEBOOK'
);

ALTER TABLE users
ALTER COLUMN creation_type TYPE user_creation_type USING creation_type::user_creation_type;

ALTER TABLE users ALTER COLUMN creation_type SET DEFAULT 'REGISTRATION';
ALTER TABLE users ALTER COLUMN creation_type SET NOT NULL;