SET search_path = auth, pg_catalog;

ALTER TABLE users ALTER COLUMN password DROP NOT NULL;
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users ADD COLUMN creation_type VARCHAR(44);
ALTER TABLE users ADD COLUMN facebook_id VARCHAR(64);