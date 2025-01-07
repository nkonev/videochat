ALTER TABLE chat ADD COLUMN regular_participant_can_write_message BOOLEAN NOT NULL DEFAULT TRUE;

DROP FUNCTION IF EXISTS CREATE_CHAT(IN chat_name TEXT, IN tet_a_tet BOOLEAN, IN can_resend BOOLEAN, IN available_to_search BOOLEAN, IN blog BOOLEAN, IN regular_participant_can_publish_message BOOLEAN, IN regular_participant_can_pin_message BOOLEAN, IN blog_about BOOLEAN);

-- redefine CREATE_CHAT
CREATE OR REPLACE FUNCTION CREATE_CHAT(IN chat_name TEXT, IN tet_a_tet BOOLEAN DEFAULT FALSE, IN can_resend BOOLEAN DEFAULT FALSE, IN available_to_search BOOLEAN DEFAULT FALSE, IN blog BOOLEAN DEFAULT FALSE, IN regular_participant_can_publish_message BOOLEAN DEFAULT FALSE, IN regular_participant_can_pin_message BOOLEAN DEFAULT FALSE, IN blog_about BOOLEAN DEFAULT FALSE, IN regular_participant_can_write_message BOOLEAN DEFAULT TRUE) RETURNS RECORD AS $$
DECLARE
    chat_id BIGINT;
    chat_last_update_date_time TIMESTAMP;
    query1 TEXT;
    ret RECORD;
BEGIN
    -- insert into chat table
    INSERT INTO chat(title, tet_a_tet, can_resend, available_to_search, blog, regular_participant_can_publish_message, regular_participant_can_pin_message, blog_about, regular_participant_can_write_message)
    VALUES(chat_name, tet_a_tet, can_resend, available_to_search, blog, regular_participant_can_publish_message, regular_participant_can_pin_message, blog_about, regular_participant_can_write_message)
    RETURNING id, last_update_date_time INTO chat_id, chat_last_update_date_time;

    -- create message table
    query1 := format('CREATE TABLE %s() INHERITS (message)', 'message_chat_' || chat_id);
    EXECUTE query1;
    query1 := format('ALTER TABLE %s ADD PRIMARY KEY(id)', 'message_chat_' || chat_id);
    EXECUTE query1;
    query1 := format('CREATE SEQUENCE %s OWNED BY %s START 1;', 'message_chat_id_' || chat_id, 'message_chat_' || chat_id || '.id');
    EXECUTE query1;
    query1 := format('ALTER TABLE %s ALTER COLUMN id SET DEFAULT nextval(''%s'');', 'message_chat_' || chat_id, 'message_chat_id_' || chat_id);
    EXECUTE query1;

    -- create reaction table
    query1 := format('CREATE TABLE %s() INHERITS (message_reaction)', 'message_reaction_chat_' || chat_id);
    EXECUTE query1;
    query1 := format('ALTER TABLE %s ADD PRIMARY KEY(user_id, message_id, reaction)', 'message_reaction_chat_' || chat_id);
    EXECUTE query1;
    query1 := format('ALTER TABLE %s ADD FOREIGN KEY(message_id) REFERENCES %s ON DELETE CASCADE;', 'message_reaction_chat_' || chat_id, 'message_chat_' || chat_id || '(id)');
    EXECUTE query1;

    SELECT chat_id, chat_last_update_date_time INTO ret;
    RETURN ret;
END
$$ LANGUAGE plpgsql;
