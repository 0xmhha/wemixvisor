.PHONY: all build install clean clean-all test test-e2e test-all coverage fmt lint

GO_VERSION_DIR ?= $(HOME)/.gvm/gos/go1.23.11
GO_BIN_DIR := $(GO_VERSION_DIR)/bin
GO_CMD := $(GO_BIN_DIR)/go
ifeq (,$(wildcard $(GO_CMD)))
GO_CMD := go
endif
GO_ENV := PATH=$(GO_BIN_DIR):$$PATH

VERSION := 0.1.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X github.com/wemix/wemixvisor/internal/commands.Version=$(VERSION) \
           -X github.com/wemix/wemixvisor/internal/commands.GitCommit=$(COMMIT) \
           -X github.com/wemix/wemixvisor/internal/commands.BuildDate=$(BUILD_DATE)

all: build

build:
	@echo "Building wemixvisor..."
	@$(GO_ENV) $(GO_CMD) build -ldflags "$(LDFLAGS)" -o bin/wemixvisor ./cmd/wemixvisor

install:
	@echo "Installing wemixvisor..."
	@$(GO_ENV) $(GO_CMD) install -ldflags "$(LDFLAGS)" ./cmd/wemixvisor

clean:
	@echo "Cleaning binaries..."
	@rm -rf bin/
	@rm -f wemixvisor

clean-all: clean
	@echo "Cleaning coverage artifacts..."
	@rm -f coverage.out coverage.html

test:
	@echo "Running unit tests..."
	@$(GO_ENV) $(GO_CMD) test -v -race -coverprofile=coverage.out ./...
	@echo "Test coverage:"
	@$(GO_ENV) $(GO_CMD) tool cover -func=coverage.out

test-e2e:
	@echo "Running e2e tests..."
	@$(GO_ENV) $(GO_CMD) test -v -tags=e2e ./test/e2e/...

test-all: test test-e2e

coverage:
	@echo "Generating coverage report..."
	@$(GO_ENV) $(GO_CMD) test -v -race -coverprofile=coverage.out ./...
	@$(GO_ENV) $(GO_CMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

fmt:
	@echo "Formatting code..."
	@$(GO_ENV) $(GO_CMD) fmt ./...

lint:
	@echo "Running linter..."
	@$(GO_ENV) golangci-lint run ./...

.DEFAULT_GOAL := build
