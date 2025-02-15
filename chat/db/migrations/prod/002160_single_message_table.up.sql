ALTER TABLE message ADD COLUMN chat_id BIGINT NOT NULL;
SELECT create_distributed_table('message', 'chat_id');

ALTER TABLE message_reaction ADD COLUMN chat_id BIGINT NOT NULL;
SELECT create_distributed_table('message_reaction', 'chat_id');

ALTER TABLE chat ADD COLUMN last_generated_message_id BIGINT NOT NULL DEFAULT 0;

DROP FUNCTION IF EXISTS CREATE_CHAT(IN chat_name TEXT, IN tet_a_tet BOOLEAN, IN can_resend BOOLEAN, IN available_to_search BOOLEAN, IN blog BOOLEAN, IN regular_participant_can_publish_message BOOLEAN, IN regular_participant_can_pin_message BOOLEAN, IN blog_about BOOLEAN, IN regular_participant_can_write_message BOOLEAN);
DROP PROCEDURE IF EXISTS DELETE_CHAT(IN chat_id BIGINT);
