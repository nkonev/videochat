-- DROP TABLE IF EXISTS databasechangelog; -- for copy-paste purposes, actually we shouldn't remove it during migration

DROP TABLE IF EXISTS user_account CASCADE;
DROP TABLE IF EXISTS user_settings CASCADE;
DROP TYPE IF EXISTS user_creation_type CASCADE;
DROP TYPE IF EXISTS user_role CASCADE;
DROP FUNCTION IF EXISTS utc_now CASCADE;
