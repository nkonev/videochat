.PHONY: download

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

package:
	CGO_ENABLED=0 go build -trimpath -ldflags '-w -extldflags "-static"'