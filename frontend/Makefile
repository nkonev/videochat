.PHONY: check-env download package-docker push-docker clean clean-cache

download:
	npm install

check-env:
	docker version && echo -n 'npm: ' && npm --version && echo -n 'node: ' && node --version

BUILDDIR := ./build
DIST := ./dist
GIT_COMMIT := $(shell git rev-list -1 HEAD)
STATIC_JSON := $(DIST)/git.json

generate-git:
	echo "{\"commit\": \"$(GIT_COMMIT)\", \"microservice\": \"frontend\"}" > $(STATIC_JSON)

IMAGE := nkonev/chat-frontend:latest

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
	echo "Will push docker frontend image" && \
	docker push $(IMAGE)

clean:
	rm -rf $(DIST) $(BUILDDIR)

# https://vitejs.dev/guide/dep-pre-bundling.html
clean-cache:
	rm -rf node_modules/.vite node_modules/.cache

