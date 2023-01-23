ALTER TABLE message DROP COLUMN chat_id;


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
