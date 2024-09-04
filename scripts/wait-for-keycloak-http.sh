#!/usr/bin/env bash

KEYCLOAK_HOST_PORT="$1"
MAX_ATTEMPTS="$2"

echo 'Waiting for Keycloak HTTP port'

set +e

for ((n=0; n<=$MAX_ATTEMPTS; n++))
do
  echo "attempt $n / $MAX_ATTEMPTS"

  curl -Ss -f http://${KEYCLOAK_HOST_PORT}/health/ready > /dev/null

  if [[ "$?" == "0" ]]; then
    echo 'Keycloak HTTP port is available'
    exit 0
  fi

  sleep 1
done

echo "Time is out"
exit 1
