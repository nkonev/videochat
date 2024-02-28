DROP PROCEDURE IF EXISTS DELETE_CHAT(IN chat_id BIGINT);

CREATE OR REPLACE PROCEDURE DELETE_CHAT(IN chat_id BIGINT) AS $$
DECLARE
    query1 TEXT;
BEGIN
    query1 := format('DROP TABLE message_reaction_chat_%s;', chat_id);
    EXECUTE query1;
    query1 := format('DROP TABLE message_chat_%s;', chat_id);
    EXECUTE query1;
    query1 := format('DELETE FROM chat WHERE id = %s;', chat_id);
    EXECUTE query1;
END
$$ LANGUAGE plpgsql;
