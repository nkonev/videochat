-- ALTER SYSTEM SET max_connections = 400;
-- Uncomment if you need to view the full postgres logs (SQL statements, ...) via `docker logs -f postgresql-test`
-- ALTER SYSTEM SET log_statement = 'mod';
-- ALTER SYSTEM SET synchronous_commit = 'off'; -- https://postgrespro.ru/docs/postgrespro/9.5/runtime-config-wal.html#GUC-SYNCHRONOUS-COMMIT
-- ALTER SYSTEM SET shared_buffers='512MB';
-- ALTER SYSTEM SET fsync=FALSE;
-- ALTER SYSTEM SET full_page_writes=FALSE;
-- ALTER SYSTEM SET commit_delay=100000;
-- ALTER SYSTEM SET commit_siblings=10;
-- ALTER SYSTEM SET work_mem='50MB';
ALTER SYSTEM SET work_mem='512MB';
ALTER SYSTEM SET random_page_cost=1.1; -- for ssd
ALTER SYSTEM SET log_line_prefix = '%a %u@%d ';

-- https://docs.citusdata.com/en/stable/admin_guide/cluster_management.html#create-db
create database chat;
\connect chat;
CREATE EXTENSION citus;

-- https://stackoverflow.com/questions/70477676/citus-rebalance-table-shards-fe-sendauth-no-password-supplied/70494597#70494597
-- https://postgrespro.ru/docs/enterprise/16/citus
INSERT INTO pg_dist_authinfo(nodeid, rolename, authinfo) VALUES
    (0, 'postgres', 'password=postgresqlPassword');
