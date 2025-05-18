# Add
# export CONNECT_LINE=user@api.site.local
# before calling goals which deploy image to server

.PHONY: check-env download package-docker test push-docker-image-to-registry clean push-docker-image-to-server deploy-docker-image run run-bin infra infra-test

BUILDDIR := ./build
EXECUTABLE := notification
IMAGE := nkonev/chat-notification:changing
IMAGE_TO_PUBLISH := nkonev/chat-notification:latest
# should match to compose file from deploy dir
SERVICE_NAME := notification
CI_TO_COPY_DIR := /tmp/to-copy
SERVER_TO_COPY_DIR := /tmp/to-deploy/$(SERVICE_NAME)
SSH_OPTIONS := -o BatchMode=yes -o StrictHostKeyChecking=no
SERVER_COMPOSE_DIR := /opt/videochat
STACK_NAME := VIDEOCHATSTACK

download:
	echo "Nothing to download"

check-env:
	docker version && go env

generate: generate-git

GIT_COMMIT := $(shell git rev-list -1 HEAD)
STATIC_JSON := ./handlers/static/git.json

generate-git:
	echo "{\"commit\": \"$(GIT_COMMIT)\", \"microservice\": \"$(SERVICE_NAME)\"}" > $(STATIC_JSON)

test:
	go test ./... -count=1 -test.v -test.timeout=20s -p 1

package-go:
	CGO_ENABLED=0 go build -trimpath -ldflags '-w -extldflags "-static"'

package-docker:
	mkdir -p $(BUILDDIR) && \
	cp $(EXECUTABLE) $(BUILDDIR) && cp ./Dockerfile $(BUILDDIR) && \
	echo "Will build docker $(SERVICE_NAME) image" && \
 	docker build -t $(IMAGE) $(BUILDDIR)

package: package-go package-docker

push-docker-image-to-registry:
	echo "Will push docker $(SERVICE_NAME) image" && \
	docker tag $(IMAGE) $(IMAGE_TO_PUBLISH) && \
	docker push $(IMAGE_TO_PUBLISH)

push-docker-image-to-server:
	echo "Will push docker $(SERVICE_NAME) image directly on the server"
	mkdir -p $(CI_TO_COPY_DIR)
	docker save $(IMAGE) -o $(CI_TO_COPY_DIR)/$(SERVICE_NAME).tar
	ssh $(SSH_OPTIONS) -q ${CONNECT_LINE} 'docker service rm $(STACK_NAME)_$(SERVICE_NAME) ; rm -rf $(SERVER_TO_COPY_DIR) ; mkdir -p $(SERVER_TO_COPY_DIR) && echo "dir created"'
	scp $(CI_TO_COPY_DIR)/$(SERVICE_NAME).tar ${CONNECT_LINE}:$(SERVER_TO_COPY_DIR)
	ssh $(SSH_OPTIONS) -q ${CONNECT_LINE} 'docker load -i $(SERVER_TO_COPY_DIR)/$(SERVICE_NAME).tar ; rm -rf $(SERVER_TO_COPY_DIR)'

deploy-docker-image:
	ssh $(SSH_OPTIONS) -q ${CONNECT_LINE} 'docker stack deploy --compose-file $(SERVER_COMPOSE_DIR)/docker-compose-$(SERVICE_NAME).yml $(STACK_NAME)'

clean:
	rm -rf $(STATIC_JSON) $(EXECUTABLE) $(BUILDDIR)

run-bin:
	./$(EXECUTABLE)

run: check-env download generate package-go infra
	./$(EXECUTABLE)

infra:
	docker compose -f ../docker-compose.yml up -d postgresql rabbitmq jaeger
	../scripts/wait-for-it.sh -t 30 127.0.0.1:35432 -- echo 'postgres is up'
	../scripts/wait-for-it.sh -t 30 127.0.0.1:36672 -- echo 'rabbitmq is up'
	../scripts/wait-for-http.sh 'localhost:35672' 60 '' 'RabbitMQ' # wait for rabbitmq http port will be available
	../scripts/wait-for-it.sh -t 30 127.0.0.1:36686 -- echo 'jaeger web ui is up'

infra-test:
	echo "No test infra"
