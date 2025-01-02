#!/bin/bash
set -eo pipefail

export OPENSEARCH_HOME=/usr/share/opensearch
INIT_COMPLETED="$OPENSEARCH_HOME/data/.init_completed"
ORIGINAL_SCRIPT="$OPENSEARCH_HOME/opensearch-docker-entrypoint.sh"
INITD=/docker-entrypoint-init.d
INIT_HOST=localhost
INIT_PORT=9199
MAX_ATTEMPTS=360

wait_for_start() {
  local WEB_HOST_PORT=http://$1:$2/_cluster/health

  echo "Waiting for HTTP port by $WEB_HOST_PORT"

  local wait_successful=false

  set +e

  for ((n=0; n<=$MAX_ATTEMPTS; n++)); do
    echo "attempt $n / $MAX_ATTEMPTS"

    curl -Ss -f "$WEB_HOST_PORT" &> /dev/null

    if [[ "$?" == "0" ]]; then
        wait_successful=true
        break
    fi

    sleep 1
  done

  set -e

  if [ "$wait_successful" = true ] ; then
      echo 'HTTP port is available'
      return 0
  else
      echo "Time is out"
      return 1
  fi
}

wait_for_stop() {
  echo "Waiting for process $1 exit"
  tail --pid=$1 -f /dev/null
  echo "Process $1 has finished"
}

docker_process_init_files() {
	echo "Running init scripts"
	local f
	for f; do
		case "$f" in
			*.sh)
				# https://github.com/docker-library/postgres/issues/450#issuecomment-393167936
				# https://github.com/docker-library/postgres/pull/452
				if [ -x "$f" ]; then
					echo "$0: running $f"
					"$f" $INIT_HOST $INIT_PORT
				else
					echo "$0: sourcing $f"
					. "$f" $INIT_HOST $INIT_PORT
				fi
				;;
			*)         echo "$0: ignoring $f" ;;
		esac
		echo "Finished init scripts"
	done
}

if [[ ! -f $INIT_COMPLETED && -d "$INITD" && "$(ls -A $INITD)" ]]; then
    echo "Init run on local address"
    env "http.port=$INIT_PORT" $ORIGINAL_SCRIPT "$@" &
    pid=$!

    echo "Init process pid is $pid"

    wait_for_start $INIT_HOST $INIT_PORT

    docker_process_init_files $INITD/*
    touch $INIT_COMPLETED

    kill $pid

    wait_for_stop $pid

    echo "Normal run after init"
    exec $ORIGINAL_SCRIPT "$@"
else
    echo "Normal run without init"
    exec $ORIGINAL_SCRIPT "$@"
fi
