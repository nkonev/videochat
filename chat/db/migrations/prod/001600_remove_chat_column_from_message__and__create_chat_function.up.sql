ALTER TABLE message DROP COLUMN chat_id;


CREATE OR REPLACE FUNCTION CREATE_CHAT(IN chat_name TEXT, IN tet_a_tet BOOLEAN DEFAULT false) RETURNS RECORD AS $$
DECLARE
    chat_id BIGINT;
    chat_last_update_date_time TIMESTAMP;
    query1 text;
    ret RECORD;
BEGIN
    INSERT INTO chat(title, tet_a_tet) VALUES(chat_name, tet_a_tet) RETURNING id, last_update_date_time INTO chat_id, chat_last_update_date_time;
    query1 := format('CREATE TABLE %s() INHERITS (message)', 'message_chat_' || chat_id);
    EXECUTE query1;
    query1 := format('ALTER TABLE %s ADD PRIMARY KEY(id);', 'message_chat_' || chat_id);
    EXECUTE query1;
    SELECT chat_id, chat_last_update_date_time INTO ret;
    RETURN ret;
END
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION UNREAD_MESSAGES(IN chat_id BIGINT, IN user_id BIGINT) RETURNS BIGINT AS $$
DECLARE
    messages_table_exists bool;
    query1 text;
    res bigint;
BEGIN
    SELECT EXISTS (
                   SELECT FROM information_schema.tables
                   WHERE table_name   = 'message_chat_' || chat_id
               ) INTO messages_table_exists;
    IF messages_table_exists = true
    THEN
        query1 := format('SELECT COUNT(*) FROM message_chat_%s WHERE id > COALESCE((SELECT last_message_id FROM message_read WHERE user_id = %s AND chat_id = %s), 0);', chat_id, user_id, chat_id);
        EXECUTE query1 INTO res;
    ELSE
        res := 0;
    END IF;
    RETURN res;
END
$$ LANGUAGE plpgsql;
