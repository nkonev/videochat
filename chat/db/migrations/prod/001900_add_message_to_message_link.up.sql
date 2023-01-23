ALTER TABLE message
    ADD COLUMN embed_message_id BIGINT,
    ADD COLUMN embed_chat_id BIGINT,
    ADD COLUMN embed_owner_id BIGINT,
    ADD COLUMN embed_message_type VARCHAR(16);

alter table chat add column can_resend boolean not null default false;

CREATE OR REPLACE FUNCTION CREATE_CHAT(IN chat_name TEXT, IN tet_a_tet BOOLEAN DEFAULT false, in can_resend BOOLEAN DEFAULT false) RETURNS RECORD AS $$
DECLARE
    chat_id BIGINT;
    chat_last_update_date_time TIMESTAMP;
    query1 text;
    ret RECORD;
BEGIN
    INSERT INTO chat(title, tet_a_tet, can_resend) VALUES(chat_name, tet_a_tet, can_resend) RETURNING id, last_update_date_time INTO chat_id, chat_last_update_date_time;
    query1 := format('CREATE TABLE %s() INHERITS (message)', 'message_chat_' || chat_id);
    EXECUTE query1;
    query1 := format('ALTER TABLE %s ADD PRIMARY KEY(id);', 'message_chat_' || chat_id);
    EXECUTE query1;
    SELECT chat_id, chat_last_update_date_time INTO ret;
    RETURN ret;
END
$$ LANGUAGE plpgsql;
