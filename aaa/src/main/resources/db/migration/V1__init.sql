
SET search_path = auth, pg_catalog;

CREATE TYPE user_role AS ENUM (
    'ROLE_ADMIN',
    'ROLE_MODERATOR',
    'ROLE_USER'
);

-- User Schema
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
	username VARCHAR(50) NOT NULL UNIQUE,
	password VARCHAR(100) NOT NULL,
	avatar VARCHAR(256),
	enabled BOOLEAN NOT NULL DEFAULT TRUE,
	expired BOOLEAN NOT NULL DEFAULT FALSE,
	locked BOOLEAN NOT NULL DEFAULT FALSE,
	email VARCHAR(100) NOT NULL,
	role auth.user_role NOT NULL DEFAULT 'ROLE_USER'
);


/*
-- Persistent Login (Remember-Me)
CREATE TABLE persistent_logins (
	username VARCHAR(64) NOT NULL,
	series VARCHAR(64) PRIMARY KEY,
	token VARCHAR(64) NOT NULL,
	last_used TIMESTAMP NOT NULL
);
*/

SET search_path = public, pg_catalog;


CREATE TABLE images.user_avatar_image (
	id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
	img BYTEA,
	content_type VARCHAR(64),
	create_date_time timestamp NOT NULL DEFAULT (now() at time zone 'utc')
);



UPDATE auth.users SET role = 'ROLE_ADMIN' WHERE id = (SELECT id FROM auth.users WHERE username = 'admin');
