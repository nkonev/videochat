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
