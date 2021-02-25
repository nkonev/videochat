.PHONY: test push-docker clean

check-env:
	docker version && go env

generate: generate-git generate-static-go

GIT_COMMIT := $(shell git rev-list -1 HEAD)
STATIC_JSON := ./static/git.json
STATIC_ASSETS_PACKAGE := static_assets

generate-git:
	echo "{\"commit\": \"$(GIT_COMMIT)\", \"microservice\": \"video\"}" > ${STATIC_JSON}

generate-static-go:
	statik -src=./static -p ${STATIC_ASSETS_PACKAGE} -ns assets -f && echo 'completed generating embed'

test:
	go test ./... -count=1 -test.v -test.timeout=20s -p 1

package-go:
	CGO_ENABLED=0 go build -trimpath -ldflags '-w -extldflags "-static"'

BUILDDIR := ./build
EXECUTABLE := video
IMAGE = nkonev/chat-video:latest

package-docker:
	mkdir -p $(BUILDDIR) && \
	cp $(EXECUTABLE) $(BUILDDIR) && cp ./Dockerfile $(BUILDDIR) && \
	echo "Will build docker video image" && \
 	docker build --build-arg BINARY=$(EXECUTABLE) -t $(IMAGE) $(BUILDDIR)

package: package-go package-docker

push-docker:
	echo "Will push docker video image" && \
	docker push $(IMAGE)

clean:
	rm -rf ${STATIC_JSON} "./${STATIC_ASSETS_PACKAGE}" ${EXECUTABLE} ${BUILDDIR}