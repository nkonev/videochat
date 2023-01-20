ALTER TABLE message
    ADD COLUMN embed_message_id BIGINT,
    ADD COLUMN embed_chat_id BIGINT,
    ADD COLUMN embed_owner_id BIGINT,
    ADD COLUMN embed_message_type VARCHAR(16);
