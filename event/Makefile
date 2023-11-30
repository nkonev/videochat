.PHONY: check-env download package-docker test push-docker clean install-graphql

download:
	echo "Nothing to download"

check-env:
	docker version && go env

generate: generate-git

GIT_COMMIT := $(shell git rev-list -1 HEAD)
STATIC_JSON := ./handlers/static/git.json

generate-git:
	echo "{\"commit\": \"$(GIT_COMMIT)\", \"microservice\": \"event\"}" > $(STATIC_JSON)

install-graphql:
	go install github.com/99designs/gqlgen@v0.17.20

generate-graphql:
	gqlgen generate

test:
	go test ./... -count=1 -test.v -test.timeout=20s -p 1

package-go:
	CGO_ENABLED=0 go build -trimpath -ldflags '-w -extldflags "-static"'

BUILDDIR := ./build
EXECUTABLE := event
IMAGE := nkonev/event:latest

package-docker:
	mkdir -p $(BUILDDIR) && \
	cp $(EXECUTABLE) $(BUILDDIR) && cp ./Dockerfile $(BUILDDIR) && \
	echo "Will build docker event image" && \
 	docker build -t $(IMAGE) $(BUILDDIR)

package: package-go package-docker

push-docker:
	echo "Will push docker event image" && \
	docker push $(IMAGE)

clean:
	rm -rf $(STATIC_JSON) $(EXECUTABLE) $(BUILDDIR)
