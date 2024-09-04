ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET log_statement = 'all';
-- ALTER SYSTEM SET synchronous_commit = 'off'; -- https://postgrespro.ru/docs/postgrespro/9.5/runtime-config-wal.html#GUC-SYNCHRONOUS-COMMIT
-- ALTER SYSTEM SET shared_buffers='512MB';
-- ALTER SYSTEM SET fsync=FALSE;
-- ALTER SYSTEM SET full_page_writes=FALSE;
-- ALTER SYSTEM SET commit_delay=100000;
-- ALTER SYSTEM SET commit_siblings=10;
ALTER SYSTEM SET work_mem='512MB';
ALTER SYSTEM SET random_page_cost=1.1; -- for ssd
ALTER SYSTEM SET log_line_prefix = '%a %u@%d ';

create user aaa with password 'aaaPazZw0rd';
create database aaa with owner aaa;
\connect aaa;

create user chat with password 'chatPazZw0rd';
create database chat with owner chat;
\connect chat;

create user notification with password 'notificationPazZw0rd';
create database notification with owner notification;
\connect notification;

create user video with password 'videoPazZw0rd';
create database video with owner video;
\connect video;
