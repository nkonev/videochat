.PHONY: download test push-docker

download: download-statik download-proto-compiler

download-statik:
	go get github.com/rakyll/statik

download-proto-compiler:
	docker pull znly/protoc:0.4.0

check-env:
	docker version && go env

generate: generate-git generate-static-go generate-protobufs

GIT_COMMIT := $(shell git rev-list -1 HEAD)

generate-git:
	echo "{\"commit\": \"$(GIT_COMMIT)\", \"microservice\": \"chat\"}" > ./static/git.json

generate-static-go:
	statik -src=./static -p static_assets -ns assets -f && statik -src=./db/migrations -p static_migrations -ns migrations -dest ./db -f && echo 'completed generating embed'

PROTOBUF_SRC_DIR = $(shell cd ..; pwd)

generate-protobufs:
	mkdir -p ./proto && \
	docker run -it --rm -v $(PROTOBUF_SRC_DIR):/ws -w /ws znly/protoc:0.4.0 --go_out=plugins=grpc:chat/proto -I./protobuf ./protobuf/$(notdir $(wildcard ../protobuf/*.proto))

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
 	docker build --build-arg BINARY=$(EXECUTABLE) -t $(IMAGE) $(BUILDDIR)

package: package-go package-docker

push-docker:
	echo "Will login to docker" && \
	docker login -u="${DOCKER_LOGIN}" -p="${DOCKER_PASSWORD}" && \
	echo "Will push docker image" && \
	docker push $(IMAGE)