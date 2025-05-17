.PHONY: infra

infra:
	docker compose up -d
	./scripts/wait-for-it.sh -t 30 127.0.0.1:35432 -- echo 'postgres is up'
	./scripts/wait-for-it.sh -t 30 127.0.0.1:45401 -- echo 'postgresql-citus-coordinator-1 is up'
	./scripts/wait-for-it.sh -t 30 127.0.0.1:36379 -- echo 'redis is up'
	./scripts/wait-for-it.sh -t 30 127.0.0.1:36672 -- echo 'rabbitmq is up'
	./scripts/wait-for-http.sh 'localhost:35672' 60 '' 'RabbitMQ' # wait for rabbitmq http port will be available
	./scripts/wait-for-it.sh -t 30 127.0.0.1:36686 -- echo 'jaeger web ui is up'
	./scripts/wait-for-it.sh -t 30 127.0.0.1:39000 -- echo 'minio is up'
	./scripts/wait-for-it.sh -t 30 127.0.0.1:8081 -- echo 'traefik is up'

