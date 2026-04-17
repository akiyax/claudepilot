.PHONY: build test lint clean cross-compile run

# Build variables
BINARY_NAME=claudepilot
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)"

# Go files
GO_FILES=$(shell find . -name '*.go' -not -path './vendor/*')

## build: Build the daemon binary
build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/claudepilot/

## run: Build and run the daemon
run: build
	./bin/$(BINARY_NAME) --verbose

## test: Run all tests
test:
	go test -v -race ./...

## lint: Run linters
lint:
	go vet ./...
	gofmt -l $(GO_FILES) | grep -q . && exit 1 || true

## clean: Remove build artifacts
clean:
	rm -rf bin/

## cross-compile: Build for all platforms
cross-compile:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/claudepilot/
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/claudepilot/
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/claudepilot/
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/claudepilot/

## deps: Download dependencies
deps:
	go mod download

## tidy: Tidy modules
tidy:
	go mod tidy

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
