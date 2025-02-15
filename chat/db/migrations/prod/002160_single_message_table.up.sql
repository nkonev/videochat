ALTER TABLE chat ADD COLUMN last_generated_message_id BIGINT NOT NULL DEFAULT 0;
DROP FUNCTION IF EXISTS CREATE_CHAT(IN chat_name TEXT, IN tet_a_tet BOOLEAN, IN can_resend BOOLEAN, IN available_to_search BOOLEAN, IN blog BOOLEAN, IN regular_participant_can_publish_message BOOLEAN, IN regular_participant_can_pin_message BOOLEAN, IN blog_about BOOLEAN, IN regular_participant_can_write_message BOOLEAN);
DROP PROCEDURE IF EXISTS DELETE_CHAT(IN chat_id BIGINT);

ALTER TABLE message ADD COLUMN chat_id BIGINT NOT NULL;
ALTER TABLE message_reaction ADD COLUMN chat_id BIGINT NOT NULL;

alter table message drop constraint message_file_item_uuid_key;
alter table message_reaction drop CONSTRAINT message_reaction_message_id_fkey;
alter table message drop constraint message_pkey;
alter table message add primary key (chat_id, id);

SELECT create_distributed_table('message', 'chat_id');

-- TODO foreign key message_reaction -> message with colocation

-- TODO fixme
-- SELECT create_distributed_table('message_reaction', 'chat_id');
