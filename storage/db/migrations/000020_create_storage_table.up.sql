CREATE TYPE storage_type AS ENUM (
    'AVATAR',
    'FILE'
);

CREATE TABLE file_metadata(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    storage_type storage_type NOT NULL,
    file_name VARCHAR(1024) NOT NULL,
    owner_id BIGINT NOT NULL,
    create_date_time TIMESTAMP NOT NULL DEFAULT utc_now()
);