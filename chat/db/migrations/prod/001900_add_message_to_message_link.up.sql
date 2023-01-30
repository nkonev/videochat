ALTER TABLE message
    ADD COLUMN embed_message_id BIGINT,
    ADD COLUMN embed_chat_id BIGINT,
    ADD COLUMN embed_owner_id BIGINT,
    ADD COLUMN embed_message_type VARCHAR(16);

alter table chat add column can_resend boolean not null default false;
