CREATE TABLE chat_participant_notification (
    chat_id bigint NOT NULL REFERENCES chat(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    consider_messages_as_unread BOOLEAN,
    PRIMARY KEY (chat_id, user_id)
)
