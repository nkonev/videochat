DO $$
    DECLARE
        chat_id bigint;
        query1 text;
    BEGIN
        FOR chat_id IN SELECT id FROM chat
            LOOP
                query1 := format('ALTER TABLE message_chat_%s ADD COLUMN embed_message_id BIGINT, ADD COLUMN embed_message_type VARCHAR(16);', chat_id);
                EXECUTE query1;
            END LOOP;
    END
$$ LANGUAGE plpgsql;
