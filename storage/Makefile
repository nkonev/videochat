.PHONY: download

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

package:
	CGO_ENABLED=0 go build -trimpath -ldflags '-w -extldflags "-static"'