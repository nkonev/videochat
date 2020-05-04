SET search_path = auth, pg_catalog;

ALTER TABLE users ADD COLUMN vkontakte_id VARCHAR(64);
ALTER TABLE users ADD UNIQUE(vkontakte_id);