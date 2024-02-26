CREATE FUNCTION utc_now() returns TIMESTAMP AS $$ SELECT now() at time zone 'utc' $$ LANGUAGE SQL;
ALTER TABLE user_account ADD COLUMN create_date_time TIMESTAMP NOT NULL DEFAULT utc_now();
