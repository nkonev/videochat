# chat application

A (future) replacement of chat microservice in videochat project, 
built using CQRS (anti)pattern on top of Kafka as event store 
and PostgreSQL with Citus sharding extension as projection store.

See [It's Okay To Store Data In Kafka](https://www.confluent.io/blog/okay-store-data-apache-kafka/).

# Start
```bash
docker compose up -d
go run . serve | jq

# or
make package
./chat serve --server.address=:1235
./chat serve --server.address=:8082
./chat serve --server.address=:8083
```

# Play with
```bash
# create a chat
curl -i -X POST -H 'Content-Type: application/json' -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat' -d '{"name": "new chat"}'

# create a chat with extra participants
curl -i -X POST -H 'Content-Type: application/json' -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat' -d '{"name": "new chat", "participantIds":[2,3,4]}'

# rename the chat
curl -i -X PUT -H 'Content-Type: application/json' -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat' -d '{"id": 1, "name": "super new chat"}'

# show chats
curl -Ss -X GET -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat/search' | jq
# show chats with pagination
curl -Ss -X GET --url 'http://localhost:1235/api/chat/search?size=40&pinned=false&lastUpdateDateTime=2024-10-31T22:37:34.643937Z&id=477&reverse=true&includeStartingFrom=true' -H 'Accept: application/json' -H 'X-Auth-Userid: 1' | jq
curl -Ss -X GET --url 'http://localhost:1235/api/chat/search?size=40reverse=false&includeStartingFrom=true' -H 'Accept: application/json' -H 'X-Auth-Userid: 1' | jq
# search chats
curl -Ss -X GET -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat/search?searchString=new' | jq
# pin chat
curl -i -X PUT -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat/1/pin?pin=true'

# create a message
curl -i -X POST -H 'Content-Type: application/json' -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat/1/message' -d '{"text": "new message"}'
curl -i -X POST -H 'Content-Type: application/json' -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat/1/message' -d '{"text": "new message 2"}'
curl -i -X POST -H 'Content-Type: application/json' -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat/1/message' -d '{"text": "new message 3"}'

# show messages
curl -Ss -X GET -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat/1/message/search' | jq

# add a reaction
curl -i -X PUT -H 'X-Auth-Userid: 1' -H 'Content-Type: application/json' --url 'http://localhost:1235/api/chat/1/message/1/reaction' -d '{"reaction": "😀"}'

# read message
curl -i -X PUT -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat/1/message/2/read'

# add participant into chat
curl -i -X PUT -H 'X-Auth-Userid: 1' -H 'Content-Type: application/json' --url 'http://localhost:1235/api/chat/1/participant' -d '{"addParticipantIds": [2, 3]}'

# remove participant from chat
curl -i -X DELETE -H 'X-Auth-Userid: 1' -H 'Content-Type: application/json' --url 'http://localhost:1235/api/chat/1/participant/3'

# show participants
curl -Ss -X GET -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat/1/participants' | jq

# get his chats - show unreads
curl -Ss -X GET -H 'X-Auth-Userid: 2' --url 'http://localhost:1235/api/chat/search' | jq

# remove message from chat
curl -i -X DELETE  -H 'X-Auth-Userid: 1' --url 'http://localhost:1235/api/chat/1/message/1'

# show has new messages (unreads)
curl -Ss -X GET -H 'X-Auth-Userid: 2' --url 'http://localhost:1235/api/chat/has-new-messages' | jq

# read
curl -i -X PUT -H 'X-Auth-Userid: 2' --url 'http://localhost:1235/api/chat/1/message/read/500'

# ... or set to consider (contribute)
curl -i -X PUT -H 'Content-Type: application/json' -H 'X-Auth-Userid: 2' --url 'http://localhost:1235/api/chat/2/notification' -d '{"considerMessagesOfThisChatAsUnread": false}'

# make blog
curl -i -X PUT -H 'Content-Type: application/json' --url 'http://localhost:1235/api/chat' -d '{"id": 1, "name": "new chat", "blog": true}'
curl -i -X PUT --url 'http://localhost:1235/api/chat/1/message/1/blog-post'

# show blog
curl -Ss -X GET --url 'http://localhost:1235/api/blog/search' | jq
curl -Ss -X GET --url 'http://localhost:1235/api/blog/1' | jq
curl -Ss -X GET --url 'http://localhost:1235/api/blog/1/comment/search' | jq

# with correlation id
curl -i -X POST -H 'Content-Type: application/json' -H 'X-Auth-Userid: 1' -H 'X-CorrelationId: 9e49b4dd-4068-4c6a-ada0-da78f44bdeba' --url 'http://localhost:1235/api/chat' -d '{"name": "new chat"}'
# then see kafka

# reset offsets for consumer groups
go run . reset

curl -i -X DELETE --url 'http://localhost:1235/internal/truncate'
```

# Migration from old chat
```bash
make package
make infra_down
make infra
./chat migrate --performMigration=true
./chat rewind --rabbitmq.skipPublishOutputEventsOnRewind=true --rabbitmq.skipPublishNotificationEventsOnRewind=true > /tmp/chat.log
```

# Tracing
See `Trace-Id` header and put its value into [Jaeger UI](http://localhost:16686)

# Various commands
```bash
# show logs
docker compose logs -f kafka
docker compose logs -f postgresql

docker compose exec -it kafka bash
docker compose exec -it kafka /opt/kafka/bin/kafka-consumer-groups.sh --bootstrap-server kafka:29092 --list
docker compose exec -it kafka /opt/kafka/bin/kafka-consumer-groups.sh --bootstrap-server kafka:29092 --describe --group ChatProjection --offsets

# show kafka topic's messages
docker compose exec -it kafka /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server kafka:29092 --topic event-chat --from-beginning --property print.key=true --property print.headers=true

# show offsets
docker compose exec -it kafka /opt/kafka/bin/kafka-consumer-groups.sh --bootstrap-server kafka:29092 --group ChatProjection --describe

# non-actual resetting - missed fast-forwarding of sequences
docker compose exec -it kafka /opt/kafka/bin/kafka-consumer-groups.sh --bootstrap-server kafka:29092 --group ChatProjection --reset-offsets --to-earliest --execute --topic event-chat
# reset db
docker rm -f postgresql
docker volume rm go-cqrs-example_postgres_data
docker compose up -d postgresql

# export
./chat export --cqrs.export.file=/tmp/export.jsonl
# or
./chat export --cqrs.export.file=stdout > /tmp/export.jsonl

# set env
CHAT_LOGGER_LEVEL=debug ./chat serve
```

```sql
docker exec -it chat-postgresql-citus-coordinator-1-1  psql -U postgres -d chat

select citus_version();
SELECT * FROM citus_nodes;
SELECT * FROM citus_shards;
SELECT * from pg_dist_shard;
SELECT * from citus_get_active_worker_nodes();
SELECT * FROM citus_check_cluster_node_health();
```

```bash
go test -count=1 -test.v -p 1 -test.fullpath=true -timeout 30s -run ^TestPinMessage$ nkonev.name/chat/cmd
go test -count=1 -test.v -p 1 -test.fullpath=true -timeout 30s -run ^TestPublishMessage$ nkonev.name/chat/cmd
CHAT_LOGGER_LEVEL=warn CHAT_POSTGRESQL_DUMP=false CHAT_CQRS_DUMP=false CHAT_HTTP_DUMP=false CHAT_SERVER_DUMP=false CHAT_RABBITMQ_DUMPTESTACCUMULATOR=false CHAT_RABBITMQ_DUMP=false go test -count=1 -test.v -p 1 -test.fullpath=true -timeout 120s -run ^TestChatPaginate$ nkonev.name/chat/cmd
```
