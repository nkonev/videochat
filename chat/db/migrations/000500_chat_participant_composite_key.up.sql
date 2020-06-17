alter table chat_participant DROP CONSTRAINT chat_participant_chat_id_user_id_key;
ALTER TABLE chat_participant DROP COLUMN id;
ALTER TABLE chat_participant ADD PRIMARY KEY (chat_id, user_id);