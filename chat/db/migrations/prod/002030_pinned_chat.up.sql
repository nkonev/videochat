CREATE TABLE chat_pinned(
    user_id BIGINT NOT NULL,
    chat_id bigint NOT NULL REFERENCES chat(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, chat_id)
);
