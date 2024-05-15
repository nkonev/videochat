ALTER TABLE chat_participant ADD COLUMN create_date_time TIMESTAMP NOT NULL DEFAULT utc_now();
