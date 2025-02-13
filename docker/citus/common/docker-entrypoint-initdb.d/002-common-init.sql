-- ALTER SYSTEM SET max_connections = 400;
-- Uncomment if you need to view the full postgres logs (SQL statements, ...) via `docker logs -f postgresql-test`
ALTER SYSTEM SET log_statement = 'mod';
ALTER SYSTEM SET synchronous_commit = 'off'; -- https://postgrespro.ru/docs/postgrespro/9.5/runtime-config-wal.html#GUC-SYNCHRONOUS-COMMIT
-- ALTER SYSTEM SET shared_buffers='512MB';
ALTER SYSTEM SET fsync=FALSE;
ALTER SYSTEM SET full_page_writes=FALSE;
ALTER SYSTEM SET commit_delay=100000;
ALTER SYSTEM SET commit_siblings=10;
-- ALTER SYSTEM SET work_mem='50MB';
ALTER SYSTEM SET log_line_prefix = '%a %u@%d ';

-- https://docs.citusdata.com/en/stable/admin_guide/cluster_management.html#create-db
create database chat;
\connect chat;
CREATE EXTENSION citus;
