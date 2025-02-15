ALTER TABLE chat ADD COLUMN last_generated_message_id BIGINT NOT NULL DEFAULT 0;
DROP FUNCTION IF EXISTS CREATE_CHAT(IN chat_name TEXT, IN tet_a_tet BOOLEAN, IN can_resend BOOLEAN, IN available_to_search BOOLEAN, IN blog BOOLEAN, IN regular_participant_can_publish_message BOOLEAN, IN regular_participant_can_pin_message BOOLEAN, IN blog_about BOOLEAN, IN regular_participant_can_write_message BOOLEAN);
DROP PROCEDURE IF EXISTS DELETE_CHAT(IN chat_id BIGINT);

ALTER TABLE message ADD COLUMN chat_id BIGINT NOT NULL;
ALTER TABLE message_reaction ADD COLUMN chat_id BIGINT NOT NULL;

alter table message drop constraint message_file_item_uuid_key;
alter table message_reaction drop constraint message_reaction_message_id_fkey;
alter table message_reaction drop constraint message_reaction_pkey;
alter table message drop constraint message_pkey;
alter table message add primary key (chat_id, id);
alter table message_reaction add primary key (chat_id, message_id, user_id, reaction);
-- the foreign key on it is possible because message_reaction and message are colocated
alter table message_reaction add foreign key (message_id, chat_id) references message(id, chat_id) on delete cascade;
-- impossible, because chat isn't distributed
-- alter table message add foreign key (chat_id) references chat(id) on delete cascade;

SELECT create_distributed_table('message', 'chat_id');

-- https://docs.citusdata.com/en/v11.1/develop/api_udf.html#example
SELECT create_distributed_table('message_reaction', 'chat_id', colocate_with => 'message');
