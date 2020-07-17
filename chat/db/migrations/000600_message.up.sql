CREATE TABLE message (
    id bigserial primary key,
    text text NOT NULL,
    chat_id bigint NOT NULL REFERENCES chat(id) ON DELETE CASCADE,
    owner_id bigint NOT NULL,
    create_date_time TIMESTAMP NOT NULL DEFAULT utc_now(),
    edit_date_time timestamp
);

CREATE TABLE message_read (
    message_id bigint NOT NULL REFERENCES message(id) ON DELETE CASCADE,
    create_date_time TIMESTAMP NOT NULL DEFAULT utc_now(),
    user_id bigint NOT NULL, -- who have read the message
    primary key (message_id, user_id)
);