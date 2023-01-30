alter table message alter column id drop default;

DO $$
    DECLARE
        chat_id bigint;
        query1 text;
    BEGIN
        FOR chat_id IN SELECT id FROM chat
            LOOP
                query1 := format('CREATE SEQUENCE %s START 1;', 'message_chat_id_' || chat_id);
                EXECUTE query1;

                query1 := format('SELECT setval(''%s'', (SELECT COALESCE((SELECT MAX(id) FROM %s), 1)), true)', 'message_chat_id_' || chat_id, 'message_chat_' || chat_id);
                EXECUTE query1;

                query1 := format('ALTER TABLE %s ALTER COLUMN id SET DEFAULT nextval(''%s'');', 'message_chat_' || chat_id, 'message_chat_id_' || chat_id);
                EXECUTE query1;
            END LOOP;
    END
$$ LANGUAGE plpgsql;


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
    query1 := format('ALTER TABLE %s ADD PRIMARY KEY(id)', 'message_chat_' || chat_id);
    EXECUTE query1;
    query1 := format('CREATE SEQUENCE %s START 1;', 'message_chat_id_' || chat_id);
    EXECUTE query1;
    query1 := format('ALTER TABLE %s ALTER COLUMN id SET DEFAULT nextval(''%s'');', 'message_chat_' || chat_id, 'message_chat_id_' || chat_id);
    EXECUTE query1;
    SELECT chat_id, chat_last_update_date_time INTO ret;
    RETURN ret;
END
$$ LANGUAGE plpgsql;
