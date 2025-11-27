DIR := $(shell pwd)

LDFLAGS = --ldflags "\
		-X github.com/webhookx-io/webhookx/config.COMMIT=`git rev-parse --verify --short HEAD` \
		-X github.com/webhookx-io/webhookx/config.VERSION=`git tag -l --points-at HEAD | head -n 1 | sed 's/^v//'`"

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

test-unit: clean
	go test $$(go list ./... | grep -v /test/ | grep -v /examples/ ) $(FLAGS)
	cd api/license && go test $(FLAGS)

test-o11: clean
	ginkgo -r $(FLAGS) ./test/metrics ./test/tracing

test-main: clean
	ginkgo -r --skip-package=metrics,tracing $(FLAGS) ./test

goreleaser:
	goreleaser release --snapshot --clean

migrate-create:
	migrate create -ext sql -dir db/migrations -seq -digits 1 $(message)
