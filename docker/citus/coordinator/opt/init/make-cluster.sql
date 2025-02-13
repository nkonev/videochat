select citus_set_coordinator_host('postgresql-citus-coordinator-1', 5432);
select * from citus_add_node('postgresql-citus-worker-1', 5432);
select * from citus_add_node('postgresql-citus-worker-2', 5432);
