.PHONY: check-env download package-docker test push-docker clean

download:
	echo "Nothing to download"

check-env:
	docker version && java -version

generate: generate-git


generate-git:
	echo "Version is generated in pom.xml"

test:
	./mvnw test

package-java:
	./mvnw package -DskipTests

BUILDDIR := ./build
IMAGE := nkonev/chat-aaa:latest

package-docker:
	mkdir -p $(BUILDDIR) && \
	cp ./Dockerfile $(BUILDDIR) && \
	cp target/*-exec.jar $(BUILDDIR) && \
	echo "Will build docker aaa image" && \
 	docker build -t $(IMAGE) $(BUILDDIR)

package: package-java package-docker

push-docker:
	echo "Will push docker aaa image" && \
	docker push $(IMAGE)

clean:
	./mvnw clean
