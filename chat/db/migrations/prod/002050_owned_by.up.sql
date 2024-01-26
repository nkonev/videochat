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
