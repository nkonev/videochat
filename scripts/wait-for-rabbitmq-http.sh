#!/usr/bin/env bash

RABBITMQ_WEB_HOST_PORT="$1"
MAX_ATTEMPTS="$2"

echo 'Waiting for RabbitMQ HTTP port'

set +e

for ((n=0; n<=$MAX_ATTEMPTS; n++))
do
  echo "attempt $n / $MAX_ATTEMPTS"

  curl -Ss -f http://${RABBITMQ_WEB_HOST_PORT} > /dev/null

  if [[ "$?" == "0" ]]; then
    echo 'RabbitMQ HTTP port is available'
    exit 0
  fi

  sleep 1
done

echo "Time is out"
exit 1
