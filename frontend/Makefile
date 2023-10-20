.PHONY: download package-docker push-docker clean

download:
	npm install

BUILDDIR := ./build
DIST := ./dist
GIT_COMMIT := $(shell git rev-list -1 HEAD)
STATIC_JSON := ${DIST}/git.json

generate-git:
	echo "{\"commit\": \"$(GIT_COMMIT)\", \"microservice\": \"frontend\"}" > ${STATIC_JSON}

IMAGE = nkonev/chat-frontend:latest

package-node:
	npm run build

package-docker:
	mkdir -p $(BUILDDIR) && \
	cp ./Dockerfile $(BUILDDIR) && \
	cp ./nginx/frontend.conf $(BUILDDIR) && \
	mv $(DIST) $(BUILDDIR) && \
	echo "Will build docker frontend image" && \
 	docker build -t $(IMAGE) $(BUILDDIR)

package: package-node generate-git package-docker

push-docker:
	echo "Will push docker chat image" && \
	docker push $(IMAGE)

clean:
	rm -rf ${DIST} ${BUILDDIR}
