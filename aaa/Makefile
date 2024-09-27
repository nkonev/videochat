# Add
# export JAVA_HOME=/usr/lib/jvm/java-21
# before any building goal
#
# Add
# export CONNECT_LINE=user@api.site.local
# before calling goals which deploy image to server

.PHONY: check-env download package-docker test push-docker-image-to-registry clean push-docker-image-to-server deploy-docker-image run-oauth2-emu run-with-oauth2 run-with-keycloak run-demo run-with-ldap

BUILDDIR := ./build
IMAGE := nkonev/chat-aaa:changing
IMAGE_TO_PUBLISH := nkonev/chat-aaa:latest
# should match to compose file from deploy dir
SERVICE_NAME := aaa
CI_TO_COPY_DIR := /tmp/to-copy
SERVER_TO_COPY_DIR := /tmp/to-deploy/$(SERVICE_NAME)
SSH_OPTIONS := -o BatchMode=yes -o StrictHostKeyChecking=no
SERVER_COMPOSE_DIR := /opt/videochat
STACK_NAME := VIDEOCHATSTACK

download:
	./mvnw dependency:resolve

check-env:
	docker version && ${JAVA_HOME}/bin/java -version

generate: generate-git

GIT_COMMIT := $(shell git rev-list -1 HEAD)
JSON_DIR := ./target/classes/static
STATIC_JSON := $(JSON_DIR)/git.json

generate-git:
	mkdir -p $(JSON_DIR)
	echo "{\"commit\": \"$(GIT_COMMIT)\", \"microservice\": \"$(SERVICE_NAME)\"}" > $(STATIC_JSON)

test:
	./mvnw test

package-java:
	./mvnw package -DskipTests

package-docker:
	mkdir -p $(BUILDDIR) && \
	cp ./Dockerfile $(BUILDDIR) && \
	cp target/*-exec.jar $(BUILDDIR) && \
	echo "Will build docker $(SERVICE_NAME) image" && \
 	docker build -t $(IMAGE) $(BUILDDIR)

package: package-java package-docker

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
	./mvnw clean

run-oauth2-emu:
	# export JAVA_HOME=/usr/lib/jvm/java-21
	./mvnw -Poauth2_emulator spring-boot:run

run-with-oauth2:
	# export JAVA_HOME=/usr/lib/jvm/java-21
	# You need to start EmulatorServersController firstly
	${JAVA_HOME}/bin/java -jar target/aaa-0.0.0-exec.jar --spring.config.location=file:src/main/resources/config/application.yml,file:src/test/resources/config/oauth2-basic.yml,file:src/test/resources/config/user-test-controller.yml,file:src/test/resources/config/demo-migration.yml,file:src/test/resources/config/log-email.yml || true

run-with-keycloak:
	# export JAVA_HOME=/usr/lib/jvm/java-21
	${JAVA_HOME}/bin/java -jar target/aaa-0.0.0-exec.jar --spring.config.location=file:src/main/resources/config/application.yml,file:src/test/resources/config/log-email.yml,file:src/test/resources/config/oauth2-keycloak.yml --custom.schedulers.sync-keycloak.enabled=true --custom.schedulers.sync-keycloak.cron="*/5 * * * * *" || true

run-with-ldap: check-env download generate package-java
	# export JAVA_HOME=/usr/lib/jvm/java-21
	${JAVA_HOME}/bin/java -jar target/aaa-0.0.0-exec.jar --spring.config.location=file:src/main/resources/config/application.yml,file:src/test/resources/config/demo-ldap-opendj.yml || true

run-demo: check-env download generate package-java
	# export JAVA_HOME=/usr/lib/jvm/java-21
	${JAVA_HOME}/bin/java -jar target/aaa-0.0.0-exec.jar || true
