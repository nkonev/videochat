INSERT INTO chat_participant (chat_id, user_id, admin)
SELECT id AS chat_id, 1 AS user_id, TRUE FROM chat;
