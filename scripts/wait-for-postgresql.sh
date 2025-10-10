#!/usr/bin/env bash

PG_USER="$1"
DB_NAME="$2"
MAX_ATTEMPTS="$3"

echo "Waiting for $PG_USER"

set +e

function contain_string() {
      echo -n "$1" | grep "$2" &> /dev/null
      if [[ $? == 0 ]]; then
        return
      fi

      false
}

for ((n=0; n<=$MAX_ATTEMPTS; n++))
do
  echo "attempt $n / $MAX_ATTEMPTS"

  output=`docker exec postgresql-citus-coordinator-1 psql -U ${PG_USER} -d ${DB_NAME} --csv --tuples-only -c 'SELECT nodename FROM citus_nodes;'`

  if \
  contain_string "$output" 'postgresql-citus-coordinator-1' && \
  contain_string "$output" 'postgresql-citus-worker-1' && \
  contain_string "$output" 'postgresql-citus-worker-2'; then

    echo "PostgreSQL is available"
    exit 0
  fi

  sleep 1
done

echo "Time is out"
exit 1
