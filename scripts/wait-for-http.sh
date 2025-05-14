#!/usr/bin/env bash

HOST_PORT="$1"
MAX_ATTEMPTS="$2"
URL_PATH="$3"
SERVICE_NAME="$4"

echo "Waiting for $SERVICE_NAME http://${HOST_PORT}${URL_PATH}"

set +e

for ((n=0; n<=$MAX_ATTEMPTS; n++))
do
  echo "attempt $n / $MAX_ATTEMPTS"

  curl -Ss -f http://${HOST_PORT}${URL_PATH} > /dev/null

  if [[ "$?" == "0" ]]; then
    echo "$SERVICE_NAME http://${HOST_PORT}${URL_PATH} is available"
    exit 0
  fi

  sleep 1
done

echo "Time is out"
exit 1
