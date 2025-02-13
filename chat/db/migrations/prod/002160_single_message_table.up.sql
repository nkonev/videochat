ALTER TABLE message ADD COLUMN chat_id BIGINT NOT NULL;

SELECT create_distributed_table('message', 'chat_id');
