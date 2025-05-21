# Add
# export CONNECT_LINE=user@api.site.local
# before calling goals which deploy image to server

.PHONY: check-env download package-docker push-docker-image-to-registry clean clean-builddir clean-cache push-docker-image-to-server deploy-docker-image run-dev run-docker-image run infra infra-test

BUILDDIR := ./build
DIST := ./dist
GIT_COMMIT := $(shell git rev-list -1 HEAD)
IMAGE := nkonev/chat-public:changing
IMAGE_TO_PUBLISH := nkonev/chat-public:latest
# should match to compose file from deploy dir
SERVICE_NAME := public
CI_TO_COPY_DIR := /tmp/to-copy
SERVER_TO_COPY_DIR := /tmp/to-deploy/$(SERVICE_NAME)
SSH_OPTIONS := -o BatchMode=yes -o StrictHostKeyChecking=no
SERVER_COMPOSE_DIR := /opt/videochat
STACK_NAME := VIDEOCHATSTACK
JSON_DIR := $(DIST)/client
STATIC_JSON := $(JSON_DIR)/git.json

download:
	npm install

check-env:
	docker version && echo -n 'npm: ' && npm --version && echo -n 'node: ' && node --version

generate: generate-git

generate-git:
	mkdir -p $(JSON_DIR) && \
	echo "{\"commit\": \"$(GIT_COMMIT)\", \"service\": \"$(SERVICE_NAME)\"}" > $(STATIC_JSON)

package-node:
	npm run build

package-docker:
	mkdir -p $(BUILDDIR) && \
	cp ./Dockerfile $(BUILDDIR) && \
	cp ./package.json $(BUILDDIR) && \
	cp ./package-lock.json $(BUILDDIR) && \
	cp -r ./server $(BUILDDIR) && \
	cp -r ./node_modules $(BUILDDIR) && \
	cp -r ./common $(BUILDDIR) && \
	mv $(DIST) $(BUILDDIR) && \
	echo "Will build docker $(SERVICE_NAME) image" && \
 	docker build -t $(IMAGE) $(BUILDDIR)

package: package-node generate package-docker

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


clean: clean-builddir clean-cache

clean-builddir:
	rm -rf $(DIST) $(BUILDDIR)

# https://vitejs.dev/guide/dep-pre-bundling.html
clean-cache:
	rm -rf node_modules/.vite node_modules/.cache

run-dev:
	npm run dev

run-docker-image:
	docker run --rm --network=host --name chat-public $(IMAGE)

run: check-env download generate run-dev infra

infra:
	docker compose -f ../docker-compose.yml up -d traefik jaeger

infra-test:
	echo "No test infra"
