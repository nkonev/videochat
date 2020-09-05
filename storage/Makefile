.PHONY: download test

download: download-statik

download-statik:
	go get github.com/rakyll/statik

check-env:
	docker version && go env

generate: generate-git generate-static-go

GIT_COMMIT := $(shell git rev-list -1 HEAD)

generate-git:
	echo "{\"commit\": \"$(GIT_COMMIT)\", \"microservice\": \"storage\"}" > ./static/git.json

generate-static-go:
	statik -src=./static -p static_assets -ns assets -f && statik -src=./db/migrations -p static_migrations -ns migrations -dest ./db -f && echo 'completed generating embed'

test:
	go test ./... -count=1 -test.v -test.timeout=20s -p 1

package-go:
	CGO_ENABLED=0 go build -trimpath -ldflags '-w -extldflags "-static"'

BUILDDIR := ./build
EXECUTABLE := storage
IMAGE = nkonev/storage:latest

package-docker:
	mkdir -p $(BUILDDIR) && \
	cp $(EXECUTABLE) $(BUILDDIR) && cp ./Dockerfile $(BUILDDIR) && \
 	docker build --build-arg BINARY=$(EXECUTABLE) -t $(IMAGE) $(BUILDDIR)

package: package-go package-docker

push-docker:
	echo "Will login to docker" && \
	echo docker login -u="${DOCKER_LOGIN}" -p="${DOCKER_PASSWORD}" && \
	echo "Will push docker image" && \
	docker push $(IMAGE)