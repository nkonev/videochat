ALTER TABLE chat ADD COLUMN last_generated_message_id BIGINT NOT NULL DEFAULT 0;
DROP FUNCTION IF EXISTS CREATE_CHAT(IN chat_name TEXT, IN tet_a_tet BOOLEAN, IN can_resend BOOLEAN, IN available_to_search BOOLEAN, IN blog BOOLEAN, IN regular_participant_can_publish_message BOOLEAN, IN regular_participant_can_pin_message BOOLEAN, IN blog_about BOOLEAN, IN regular_participant_can_write_message BOOLEAN);

CREATE OR REPLACE PROCEDURE DELETE_CHAT(IN chat_id BIGINT) AS $$
DECLARE
    query1 TEXT;
BEGIN
    query1 := format('DELETE FROM message WHERE chat_id = %s;', chat_id);
    EXECUTE query1;
    query1 := format('DELETE FROM chat WHERE id = %s;', chat_id);
    EXECUTE query1;
END
$$ LANGUAGE plpgsql;

ALTER TABLE message ADD COLUMN chat_id BIGINT NOT NULL;
ALTER TABLE message_reaction ADD COLUMN chat_id BIGINT NOT NULL;

ALTER TABLE message DROP CONSTRAINT message_file_item_uuid_key;
ALTER TABLE message_reaction DROP CONSTRAINT message_reaction_message_id_fkey;
ALTER TABLE message_reaction DROP CONSTRAINT message_reaction_pkey;
ALTER TABLE message DROP CONSTRAINT message_pkey;
ALTER TABLE message ADD PRIMARY KEY (chat_id, id);
ALTER TABLE message_reaction ADD PRIMARY KEY (chat_id, message_id, user_id, reaction);
-- the foreign key on it is possible because message_reaction and message are colocated
ALTER TABLE message_reaction ADD FOREIGN KEY (message_id, chat_id) REFERENCES message(id, chat_id) ON DELETE CASCADE;
-- impossible, because chat isn't distributed
-- ALTER TABLE message ADD FOREIGN KEY (chat_id) REFERENCES chat(id) ON DELETE CASCADE;

SELECT create_distributed_table('message', 'chat_id');

-- https://docs.citusdata.com/en/v11.1/develop/api_udf.html#example
SELECT create_distributed_table('message_reaction', 'chat_id', colocate_with => 'message');
