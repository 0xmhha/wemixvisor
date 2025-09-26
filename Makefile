.PHONY: all build install clean test fmt lint

VERSION := 0.1.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X github.com/wemix/wemixvisor/internal/commands.Version=$(VERSION) \
           -X github.com/wemix/wemixvisor/internal/commands.GitCommit=$(COMMIT) \
           -X github.com/wemix/wemixvisor/internal/commands.BuildDate=$(BUILD_DATE)

all: build

build:
	@echo "Building wemixvisor..."
	@go build -ldflags "$(LDFLAGS)" -o bin/wemixvisor ./cmd/wemixvisor

install:
	@echo "Installing wemixvisor..."
	@go install -ldflags "$(LDFLAGS)" ./cmd/wemixvisor

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f wemixvisor

test:
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "Test coverage:"
	@go tool cover -func=coverage.out

test-e2e:
	@echo "Running e2e tests..."
	@go test -v -tags=e2e ./test/e2e/...

test-all: test test-e2e

coverage:
	@echo "Generating coverage report..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

fmt:
	@echo "Formatting code..."
	@go fmt ./...

lint:
	@echo "Running linter..."
	@golangci-lint run ./...

.DEFAULT_GOAL := build