GOPATH ?= $(shell go env GOPATH)
BUILD_DIR ?= ./build
.PHONY: build build-cw-load-test \
	build-linux build-cw-load-test-linux \
	test lint clean
.DEFAULT_GOAL := build
BUILD_FLAGS ?= -mod=readonly

build: build-cw-load-test

build-cw-load-test:
	@go build $(BUILD_FLAGS) \
		-ldflags "-X github.com/giansalex/cw-load-test/pkg/loadtest.cliVersionCommitID=`git rev-parse --short HEAD`" \
		-o $(BUILD_DIR)/cw-load-test ./cmd/cw-load-test/

build-linux: build-cw-load-test-linux

build-cw-load-test-linux:
	GOOS=linux GOARCH=amd64 $(MAKE) build-cw-load-test

test:
	go test -cover -race ./...

bench:
	go test -bench="Benchmark" -run="notests" ./...

$(GOPATH)/bin/golangci-lint:
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

lint: $(GOPATH)/bin/golangci-lint
	$(GOPATH)/bin/golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
