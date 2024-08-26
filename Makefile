LDFLAGS = --ldflags "\
		-X github.com/webhookx-io/webhookx/config.COMMIT=`git rev-parse --verify --short HEAD` \
		-X github.com/webhookx-io/webhookx/config.VERSION=`git tag -l --points-at HEAD | head -n 1`"

.PHONY: build
build:
	go build ${LDFLAGS}

.PHONY: test
test:
	go test ./...

.PHONY: test-integration
test-integration:
	echo "run integration tests"

.PHONY: test-coverage
test-coverage:
	go test ./... -coverprofile=coverage.txt

