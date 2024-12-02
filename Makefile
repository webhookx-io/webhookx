DIR := $(shell pwd)

export WEBHOOKX_TEST_OTEL_COLLECTOR_OUTPUT_PATH=$(DIR)/test/output/otel

LDFLAGS = --ldflags "\
		-X github.com/webhookx-io/webhookx/config.COMMIT=`git rev-parse --verify --short HEAD` \
		-X github.com/webhookx-io/webhookx/config.VERSION=`git tag -l --points-at HEAD | head -n 1`"

.PHONY: clean build install generate test test-coverage test-integration \
	test-integration-coverage goreleaser migrate-create test-deps

clean:
	go clean
	go clean -testcache

build:
	CGO_ENABLED=0 go build ${LDFLAGS}

install:
	go install ${LDFLAGS}

generate:
	go generate ./...

test-deps:
	mkdir -p test/output/otel
	docker compose -f test/docker-compose.yml up -d

test: clean
	go test $$(go list ./... | grep -v /test/)

test-coverage: clean
	go test $$(go list ./... | grep -v /test/) -coverprofile=coverage.txt

test-integration: clean
	go test -p 1 -v ./test/...

test-integration-coverage: clean
	go test -p 1 -v ./test/... --coverpkg ./... -coverprofile=coverage.txt

goreleaser:
	goreleaser release --snapshot --clean

migrate-create:
	migrate create -ext sql -dir db/migrations -seq -digits 1 $(message)
	