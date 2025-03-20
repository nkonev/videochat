-- ALTER SYSTEM SET max_connections = 400;
-- Uncomment if you need to view the full postgres logs (SQL statements, ...) via `docker logs -f postgresql-test`
ALTER SYSTEM SET log_statement = 'mod';
ALTER SYSTEM SET synchronous_commit = 'off'; -- https://postgrespro.ru/docs/postgrespro/9.5/runtime-config-wal.html#GUC-SYNCHRONOUS-COMMIT
-- ALTER SYSTEM SET shared_buffers='512MB';
ALTER SYSTEM SET fsync=FALSE;
ALTER SYSTEM SET full_page_writes=FALSE;
ALTER SYSTEM SET commit_delay=100000;
ALTER SYSTEM SET commit_siblings=10;
ALTER SYSTEM SET log_line_prefix = '%a %u@%d ';

-- https://docs.citusdata.com/en/v11.2/performance/performance_tuning.html
alter system set citus.max_adaptive_executor_pool_size = 1;
alter system set work_mem = '256MB';
alter system set citus.executor_slow_start_interval = 200;

-- https://docs.citusdata.com/en/stable/admin_guide/cluster_management.html#create-db
create database chat;
\connect chat;
CREATE EXTENSION citus;

-- to fix cluster restart
-- https://stackoverflow.com/questions/70477676/citus-rebalance-table-shards-fe-sendauth-no-password-supplied/70494597#70494597
-- https://postgrespro.ru/docs/enterprise/16/citus
INSERT INTO pg_dist_authinfo(nodeid, rolename, authinfo) VALUES
(0, 'postgres', 'password=postgresqlPassword');
