#!/bin/bash

set -eu

/sbin/wait-for-it.sh -t 60 postgresql-citus-worker-1:5432 -- echo 'postgresql-citus-worker-1 is up'
/sbin/wait-for-it.sh -t 60 postgresql-citus-worker-2:5432 -- echo 'postgresql-citus-worker-2 is up'

cat << EOF | psql -d chat -U postgres
SELECT citus_set_coordinator_host('postgresql-citus-coordinator-1', 5432);
SELECT * from citus_add_node('postgresql-citus-worker-1', 5432);
SELECT * from citus_add_node('postgresql-citus-worker-2', 5432);
EOF

echo "cluster is successfully set up"
