CREATE TYPE avatar_type AS ENUM (
    'AVATAR_200x200'
);

CREATE TABLE avatar_metadata(
    owner_id BIGINT NOT NULL,
    avatar_type avatar_type NOT NULL,
    file_name VARCHAR(1024) NOT NULL,
    create_date_time TIMESTAMP NOT NULL DEFAULT utc_now(),
    PRIMARY KEY (owner_id, avatar_type)
);