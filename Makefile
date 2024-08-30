LDFLAGS = --ldflags "\
		-X github.com/webhookx-io/webhookx/config.COMMIT=`git rev-parse --verify --short HEAD` \
		-X github.com/webhookx-io/webhookx/config.VERSION=`git tag -l --points-at HEAD | head -n 1`"

.PHONY: build
build:
	CGO_ENABLED=0 go build ${LDFLAGS}

.PHONY: test
test:
	go test $$(go list ./... | grep -v /test/)

.PHONY: test-coverage
test-coverage:
	go test $$(go list ./... | grep -v /test/) -coverprofile=coverage.txt

.PHONY: test-integration
test-integration:
	go test ./test/...

.PHONY: test-integration-coverage
test-integration-coverage:
	go test ./test/... --coverpkg ./... -coverprofile=coverage.txt

.PHONY: goreleaser
goreleaser:
	goreleaser release --snapshot --clean
