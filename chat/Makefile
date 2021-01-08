.PHONY: download test push-docker clean

download: download-statik download-proto-compiler

download-statik:
	go get github.com/rakyll/statik

download-proto-compiler:
	docker pull znly/protoc:0.4.0

check-env:
	docker version && go env

generate: generate-git generate-static-go generate-protobufs

GIT_COMMIT := $(shell git rev-list -1 HEAD)
STATIC_JSON := ./static/git.json
MIGRATIONS_PACKAGE := static_migrations
STATIC_ASSETS_PACKAGE := static_assets
PROTO_DIR := ./proto

generate-git:
	echo "{\"commit\": \"$(GIT_COMMIT)\", \"microservice\": \"chat\"}" > ${STATIC_JSON}

generate-static-go:
	statik -src=./static -p ${STATIC_ASSETS_PACKAGE} -ns assets -f && statik -src=./db/migrations -p ${MIGRATIONS_PACKAGE} -ns migrations -dest ./db -f && echo 'completed generating embed'

PROTOBUF_SRC_DIR = $(shell cd ..; pwd)

generate-protobufs:
	mkdir -p ${PROTO_DIR} && \
	docker run -t --rm -v $(PROTOBUF_SRC_DIR):/ws -w /ws znly/protoc:0.4.0 --go_out=plugins=grpc:chat/proto -I./protobuf ./protobuf/$(notdir $(wildcard ../protobuf/*.proto))

test:
	go test ./... -count=1 -test.v -test.timeout=20s -p 1

package-go:
	CGO_ENABLED=0 go build -trimpath -ldflags '-w -extldflags "-static"'

BUILDDIR := ./build
EXECUTABLE := chat
IMAGE = nkonev/chat:latest

package-docker:
	mkdir -p $(BUILDDIR) && \
	cp $(EXECUTABLE) $(BUILDDIR) && cp ./Dockerfile $(BUILDDIR) && \
	echo "Will build docker chat image" && \
 	docker build --build-arg BINARY=$(EXECUTABLE) -t $(IMAGE) $(BUILDDIR)

package: package-go package-docker

push-docker:
	echo "Will login to docker" && \
	docker login -u="${DOCKER_LOGIN}" -p="${DOCKER_PASSWORD}" && \
	echo "Will push docker chat image" && \
	docker push $(IMAGE)

clean:
	rm -rf ${STATIC_JSON} "./db/${MIGRATIONS_PACKAGE}" "./${STATIC_ASSETS_PACKAGE}" "${PROTO_DIR}" ${EXECUTABLE} ${BUILDDIR}