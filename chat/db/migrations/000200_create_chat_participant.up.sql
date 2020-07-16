CREATE TABLE chat_participant (
    chat_id bigint NOT NULL REFERENCES chat(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    admin BOOLEAN NOT NULL,
    PRIMARY KEY (chat_id, user_id)
)