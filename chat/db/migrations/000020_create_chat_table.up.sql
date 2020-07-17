CREATE TABLE chat(
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(1024) NOT NULL,
    create_date_time TIMESTAMP NOT NULL DEFAULT utc_now(),
    last_update_date_time TIMESTAMP NOT NULL DEFAULT utc_now()
);