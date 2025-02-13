#!/bin/bash

set -eux

/sbin/wait-for-it.sh -t 30 postgresql-citus-worker-1:5432 -- echo 'postgresql-citus-worker-1 is up'
/sbin/wait-for-it.sh -t 30 postgresql-citus-worker-2:5432 -- echo 'postgresql-citus-worker-2 is up'

cat /opt/init/make-cluster.sql | psql -U postgres

echo "cluster is successfully set up"
