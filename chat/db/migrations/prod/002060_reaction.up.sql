CREATE TABLE message_reaction(
    user_id BIGINT NOT NULL,
    reaction VARCHAR(4) NOT NULL,
    message_id BIGINT NOT NULL REFERENCES message(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, message_id, reaction)
);

-- create message_reaction_chat_N for each message table
DO $$
    DECLARE
        chat_id BIGINT;
        query1 TEXT;
    BEGIN
        FOR chat_id IN SELECT id FROM chat
            LOOP
                query1 := format('CREATE TABLE %s() INHERITS (message_reaction)', 'message_reaction_chat_' || chat_id);
                EXECUTE query1;
                query1 := format('ALTER TABLE %s ADD PRIMARY KEY(user_id, message_id, reaction)', 'message_reaction_chat_' || chat_id);
                EXECUTE query1;
                query1 := format('ALTER TABLE %s ADD FOREIGN KEY(message_id) REFERENCES %s ON DELETE CASCADE;', 'message_reaction_chat_' || chat_id, 'message_chat_' || chat_id || '(id)');
                EXECUTE query1;
            END LOOP;
    END
$$ LANGUAGE plpgsql;
