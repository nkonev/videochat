-- set owned by for older sequences
DO $$
    DECLARE
        chat_id bigint;
        query1 text;
    BEGIN
        FOR chat_id IN SELECT id FROM chat
            LOOP
                query1 := format('ALTER SEQUENCE %s OWNED BY %s;', 'message_chat_id_' || chat_id, 'message_chat_' || chat_id || '.id');
                EXECUTE query1;
            END LOOP;
    END
$$ LANGUAGE plpgsql;



DROP FUNCTION IF EXISTS CREATE_CHAT(IN chat_name TEXT, IN tet_a_tet BOOLEAN, in can_resend BOOLEAN, IN available_to_search BOOLEAN);

-- redefine CREATE_CHAT
CREATE OR REPLACE FUNCTION CREATE_CHAT(IN chat_name TEXT, IN tet_a_tet BOOLEAN DEFAULT false, in can_resend BOOLEAN DEFAULT false, IN available_to_search BOOLEAN DEFAULT false, IN blog BOOLEAN DEFAULT FALSE) RETURNS RECORD AS $$
DECLARE
    chat_id BIGINT;
    chat_last_update_date_time TIMESTAMP;
    query1 text;
    ret RECORD;
BEGIN
    INSERT INTO chat(title, tet_a_tet, can_resend, available_to_search, blog) VALUES(chat_name, tet_a_tet, can_resend, available_to_search, blog) RETURNING id, last_update_date_time INTO chat_id, chat_last_update_date_time;
    query1 := format('CREATE TABLE %s() INHERITS (message)', 'message_chat_' || chat_id);
    EXECUTE query1;
    query1 := format('ALTER TABLE %s ADD PRIMARY KEY(id)', 'message_chat_' || chat_id);
    EXECUTE query1;
    query1 := format('CREATE SEQUENCE %s OWNED BY %s START 1;', 'message_chat_id_' || chat_id, 'message_chat_' || chat_id || '.id');
    EXECUTE query1;
    query1 := format('ALTER TABLE %s ALTER COLUMN id SET DEFAULT nextval(''%s'');', 'message_chat_' || chat_id, 'message_chat_id_' || chat_id);
    EXECUTE query1;
    SELECT chat_id, chat_last_update_date_time INTO ret;
    RETURN ret;
END
$$ LANGUAGE plpgsql;
