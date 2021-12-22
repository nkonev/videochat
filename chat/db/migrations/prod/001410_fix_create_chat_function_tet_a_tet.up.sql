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
        query1 := format('ALTER TABLE %s ADD FOREIGN KEY (chat_id) REFERENCES chat(id) ON DELETE CASCADE;', 'message_chat_' || chat_id);
        EXECUTE query1;
        SELECT chat_id, chat_last_update_date_time INTO ret;
        RETURN ret;
    END
$$ LANGUAGE plpgsql;