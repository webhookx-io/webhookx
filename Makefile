LDFLAGS = --ldflags "\
		-X github.com/webhookx-io/webhookx/config.COMMIT=`git rev-parse --verify --short HEAD` \
		-X github.com/webhookx-io/webhookx/config.VERSION=`git tag -l --points-at HEAD | head -n 1`"

.PHONY: clean
clean:
	go clean
	go clean -testcache

.PHONY: build
build:
	CGO_ENABLED=0 go build ${LDFLAGS}

.PHONY: test
test: clean
	go test $$(go list ./... | grep -v /test/)

.PHONY: test-coverage
test-coverage: clean
	go test $$(go list ./... | grep -v /test/) -coverprofile=coverage.txt

.PHONY: test-integration
test-integration: clean
	go test -v ./test/...

.PHONY: test-integration-coverage
test-integration-coverage: clean
	go test -v ./test/... --coverpkg ./... -coverprofile=coverage.txt

.PHONY: goreleaser
goreleaser:
	goreleaser release --snapshot --clean
