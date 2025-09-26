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
	@echo "Running tests..."
	@go test -v ./...

fmt:
	@echo "Formatting code..."
	@go fmt ./...

lint:
	@echo "Running linter..."
	@golangci-lint run ./...

.DEFAULT_GOAL := build