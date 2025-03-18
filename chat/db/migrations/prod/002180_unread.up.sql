ALTER TABLE message_read
    DROP COLUMN last_message_id,
    ADD COLUMN unread_messages BIGINT NOT NULL DEFAULT 0;
