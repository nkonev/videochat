INSERT INTO chat_participant (chat_id, user_id, admin)
SELECT id AS chat_id, owner_id AS user_id, TRUE FROM chat;

ALTER TABLE chat DROP COLUMN owner_id;