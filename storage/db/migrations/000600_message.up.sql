CREATE TABLE message (
    id bigserial primary key,
    text text NOT NULL,
    chat_id bigint NOT NULL REFERENCES chat(id) ON DELETE CASCADE,
    owner_id bigint NOT NULL,
    create_date_time TIMESTAMP NOT NULL DEFAULT utc_now(),
    edit_date_time timestamp
);

CREATE TABLE message_read (
    last_message_id bigint NOT NULL,
    create_date_time TIMESTAMP NOT NULL DEFAULT utc_now(),
    user_id bigint NOT NULL, -- who have read the message
    chat_id bigint NOT NULL REFERENCES chat(id) ON DELETE CASCADE,
    primary key (user_id, chat_id)
);