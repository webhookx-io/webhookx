DIR := $(shell pwd)

LDFLAGS = --ldflags "\
		-X github.com/webhookx-io/webhookx/config.COMMIT=`git rev-parse --verify --short HEAD` \
		-X github.com/webhookx-io/webhookx/config.VERSION=`git tag -l --points-at HEAD | head -n 1`"

.PHONY: clean build install generate test test-coverage test-integration \
	test-integration-coverage goreleaser migrate-create test-deps

clean:
	go clean
	go clean -testcache

build:
	CGO_ENABLED=0 go build -o webhookx ${LDFLAGS} ./cmd/main

install: build
	cp webhookx $(HOME)/go/bin/webhookx

generate:
	go generate ./...

test-deps:
	mkdir -p test/output/otel
	docker compose -f test/docker-compose.yml up -d

test: clean
	go test $$(go list ./... | grep -v /test/ | grep -v /examples/ )

test-coverage: clean
	go test $$(go list ./... | grep -v /test/ | grep -v /examples/ ) -coverprofile=coverage.txt

test-integration: clean
	go test -p 1 -v ./test/...

test-integration-coverage: clean
	go test -p 1 -v ./test/... --coverpkg ./... -coverprofile=coverage.txt

goreleaser:
	goreleaser release --snapshot --clean

migrate-create:
	migrate create -ext sql -dir db/migrations -seq -digits 1 $(message)
