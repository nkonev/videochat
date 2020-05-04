SET search_path = auth, pg_catalog;

ALTER TABLE users ADD UNIQUE(email);
ALTER TABLE users ADD UNIQUE(facebook_id);
