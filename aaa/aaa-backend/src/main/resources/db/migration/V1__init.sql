
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


CREATE TABLE posts.post (
  id BIGSERIAL PRIMARY KEY,
  title CHARACTER VARYING(256) NOT NULL,
  text TEXT NOT NULL,
	text_no_tags TEXT NOT NULL,
  title_img TEXT NOT NULL,
  owner_id BIGINT NOT NULL REFERENCES auth.users(id),
  create_date_time timestamp NOT NULL DEFAULT (now() at time zone 'utc'),
  UNIQUE (title)
);

CREATE TABLE posts.comment (
  id BIGSERIAL PRIMARY KEY,
  text TEXT NOT NULL,
  post_id BIGINT NOT NULL REFERENCES posts.post(id),
  owner_id BIGINT NOT NULL REFERENCES auth.users(id),
  create_date_time timestamp NOT NULL DEFAULT (now() at time zone 'utc')
);


CREATE TABLE images.post_title_image (
	id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
	img BYTEA,
	content_type VARCHAR(64),
	create_date_time timestamp NOT NULL DEFAULT (now() at time zone 'utc')
);

CREATE TABLE images.user_avatar_image (
	id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
	img BYTEA,
	content_type VARCHAR(64),
	create_date_time timestamp NOT NULL DEFAULT (now() at time zone 'utc')
);

CREATE TABLE images.post_content_image (
	id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
	img BYTEA,
	content_type VARCHAR(64),
	create_date_time timestamp NOT NULL DEFAULT (now() at time zone 'utc')
);


CREATE INDEX title_text_idx ON posts.post USING gin (to_tsvector('russian', title || ' ' || text_no_tags));

UPDATE auth.users SET role = 'ROLE_ADMIN' WHERE id = (SELECT id FROM auth.users WHERE username = 'admin');
