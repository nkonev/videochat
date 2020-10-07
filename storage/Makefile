.PHONY: download test push-docker clean

download: download-statik

download-statik:
	go get github.com/rakyll/statik

check-env:
	docker version && go env

generate: generate-git generate-static-go

GIT_COMMIT := $(shell git rev-list -1 HEAD)
STATIC_JSON := ./static/git.json
MIGRATIONS_PACKAGE := static_migrations
STATIC_ASSETS_PACKAGE := static_assets

generate-git:
	echo "{\"commit\": \"$(GIT_COMMIT)\", \"microservice\": \"storage\"}" > ${STATIC_JSON}

generate-static-go:
	statik -src=./static -p ${STATIC_ASSETS_PACKAGE} -ns assets -f && statik -src=./db/migrations -p ${MIGRATIONS_PACKAGE} -ns migrations -dest ./db -f && echo 'completed generating embed'

test:
	go test ./... -count=1 -test.v -test.timeout=20s -p 1

package-go:
	CGO_ENABLED=0 go build -trimpath -ldflags '-w -extldflags "-static"'

BUILDDIR := ./build
EXECUTABLE := storage
IMAGE = nkonev/chat-storage:latest

package-docker:
	mkdir -p $(BUILDDIR) && \
	cp $(EXECUTABLE) $(BUILDDIR) && cp ./Dockerfile $(BUILDDIR) && \
	echo "Will build docker storage image" && \
 	docker build --build-arg BINARY=$(EXECUTABLE) -t $(IMAGE) $(BUILDDIR)

package: package-go package-docker

push-docker:
	echo "Will login to docker" && \
	docker login -u="${DOCKER_LOGIN}" -p="${DOCKER_PASSWORD}" && \
	echo "Will push docker storage image" && \
	docker push $(IMAGE)

clean:
	rm -rf ${STATIC_JSON} "./db/${MIGRATIONS_PACKAGE}" "./${STATIC_ASSETS_PACKAGE}" ${EXECUTABLE} ${BUILDDIR}