-- ALTER SYSTEM SET max_connections = 400;
-- Uncomment if you need to view the full postgres logs (SQL statements, ...) via `docker logs -f postgresql-test`
ALTER SYSTEM SET log_statement = 'all';
ALTER SYSTEM SET synchronous_commit = 'off'; -- https://postgrespro.ru/docs/postgrespro/9.5/runtime-config-wal.html#GUC-SYNCHRONOUS-COMMIT
-- ALTER SYSTEM SET shared_buffers='512MB';
ALTER SYSTEM SET fsync=FALSE;
ALTER SYSTEM SET full_page_writes=FALSE;
ALTER SYSTEM SET commit_delay=100000;
ALTER SYSTEM SET commit_siblings=10;
-- ALTER SYSTEM SET work_mem='50MB';
ALTER SYSTEM SET log_line_prefix = '%a %u@%d ';

create user aaa with password 'aaaPazZw0rd';
create database aaa with owner aaa;
\connect aaa;
-- create extension if not exists "hstore" schema pg_catalog;
-- https://www.endpoint.com/blog/2012/10/30/postgresql-autoexplain-module
-- ALTER SYSTEM set client_min_messages = notice;
-- ALTER SYSTEM set log_min_messages = notice;
-- ALTER SYSTEM set log_min_duration_statement = -1;
-- ALTER SYSTEM set log_connections = on;
-- ALTER SYSTEM set log_disconnections = on;
-- ALTER SYSTEM set log_duration = on;
--


create user chat with password 'chatPazZw0rd';
-- superuser only for test!
alter role chat superuser;
create database chat with owner chat;
\connect chat;

create user notification with password 'notificationPazZw0rd';
create database notification with owner notification;
\connect notification;

