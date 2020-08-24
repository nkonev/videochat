CREATE TYPE avatar_type AS ENUM (
    'AVATAR_200x200'
);

CREATE TABLE file_metadata(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_name VARCHAR(1024) NOT NULL,
    owner_id BIGINT NOT NULL,
    create_date_time TIMESTAMP NOT NULL DEFAULT utc_now()
);

CREATE TABLE avatar_metadata(
    owner_id BIGINT NOT NULL,
    avatar_type avatar_type NOT NULL,
    file_name VARCHAR(1024) NOT NULL,
    create_date_time TIMESTAMP NOT NULL DEFAULT utc_now(),
    PRIMARY KEY (owner_id, avatar_type)
);