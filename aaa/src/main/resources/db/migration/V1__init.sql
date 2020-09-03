
--
-- Name: user_creation_type; Type: TYPE; Schema: auth; Owner: aaa
--

CREATE TYPE user_creation_type AS ENUM (
    'REGISTRATION',
    'FACEBOOK',
    'VKONTAKTE'
);


--
-- Name: user_role; Type: TYPE; Schema: auth; Owner: aaa
--

CREATE TYPE user_role AS ENUM (
    'ROLE_ADMIN',
    'ROLE_USER'
);


--
-- Name: CAST (character varying AS user_creation_type); Type: CAST; Schema: -; Owner: -
--

CREATE CAST (character varying AS user_creation_type) WITH INOUT AS ASSIGNMENT;


--
-- Name: CAST (character varying AS user_role); Type: CAST; Schema: -; Owner: -
--

CREATE CAST (character varying AS user_role) WITH INOUT AS ASSIGNMENT;


--
-- Name: users; Type: TABLE; Schema: auth; Owner: aaa
--

CREATE TABLE users (
    id bigserial PRIMARY KEY,
    username character varying(50) UNIQUE NOT NULL,
    password character varying(100),
    avatar character varying(256),
    enabled boolean DEFAULT true NOT NULL,
    expired boolean DEFAULT false NOT NULL,
    locked boolean DEFAULT false NOT NULL,
    email character varying(100) UNIQUE,
    role user_role DEFAULT 'ROLE_USER' NOT NULL,
    creation_type user_creation_type DEFAULT 'REGISTRATION' NOT NULL,
    facebook_id character varying(64) UNIQUE,
    vkontakte_id character varying(64) UNIQUE,
    last_login_date_time timestamp without time zone
);


--
-- Data for Name: users; Type: TABLE DATA; Schema: auth; Owner: aaa
--

INSERT INTO users (id, username, password, avatar, enabled, expired, locked, email, role, creation_type, facebook_id, vkontakte_id, last_login_date_time) VALUES
 (-1, 'deleted', NULL, NULL, false, true, true, NULL, 'ROLE_USER', 'REGISTRATION', NULL, NULL, NULL),
 -- bcrypt('admin', 10)
 (1, 'admin', '$2a$10$HsyFGy9IO//nJZxYc2xjDeV/kF7koiPrgIDzPOfgmngKVe9cOyOS2', 'https://cdn3.iconfinder.com/data/icons/rcons-user-action/32/boy-512.png', true, false, false, 'admin@example.com', 'ROLE_ADMIN', 'REGISTRATION', NULL, NULL, NULL);


SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));