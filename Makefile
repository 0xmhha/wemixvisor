.PHONY: all build build-all install clean clean-all test test-unit test-integration test-e2e test-all \
        coverage coverage-html coverage-func lint fmt vet check docker docker-build docker-push \
        release help

# Go toolchain configuration
GO_VERSION_DIR ?= $(HOME)/.gvm/gos/go1.23.11
GO_BIN_DIR := $(GO_VERSION_DIR)/bin
GO_CMD := $(GO_BIN_DIR)/go
ifeq (,$(wildcard $(GO_CMD)))
GO_CMD := go
endif
GO_ENV := PATH=$(GO_BIN_DIR):$$PATH

# Version information
VERSION := 0.7.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_TAG := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build flags
LDFLAGS := -X github.com/wemix/wemixvisor/internal/cli.Version=$(VERSION) \
           -X github.com/wemix/wemixvisor/internal/cli.GitCommit=$(COMMIT) \
           -X github.com/wemix/wemixvisor/internal/cli.BuildDate=$(BUILD_DATE)

# Directories
DIST_DIR := dist
BIN_DIR := $(DIST_DIR)/bin
COVERAGE_DIR := $(DIST_DIR)/coverage
REPORTS_DIR := $(DIST_DIR)/reports
PROFILES_DIR := $(DIST_DIR)/profiles
SCRIPTS_OUTPUT_DIR := $(DIST_DIR)/scripts-output

# Binary names
BINARY_NAME := wemixvisor
BINARY_UNIX := $(BINARY_NAME)_unix
BINARY_DARWIN := $(BINARY_NAME)_darwin
BINARY_WINDOWS := $(BINARY_NAME).exe

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

# Default target
.DEFAULT_GOAL := help

##@ General

help: ## Display this help message
	@echo "$(COLOR_BOLD)Wemixvisor Makefile Commands$(COLOR_RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make $(COLOR_BLUE)<target>$(COLOR_RESET)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(COLOR_BLUE)%-20s$(COLOR_RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n$(COLOR_BOLD)%s$(COLOR_RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

all: clean build ## Clean and build binary

build: ## Build binary for current platform
	@echo "$(COLOR_GREEN)Building $(BINARY_NAME) $(VERSION)...$(COLOR_RESET)"
	@mkdir -p $(BIN_DIR)
	@$(GO_ENV) $(GO_CMD) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/wemixvisor
	@echo "$(COLOR_GREEN)✓ Binary built: $(BIN_DIR)/$(BINARY_NAME)$(COLOR_RESET)"

build-all: build-linux build-darwin build-windows ## Build binaries for all platforms

build-linux: ## Build binary for Linux
	@echo "$(COLOR_YELLOW)Building for Linux...$(COLOR_RESET)"
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=amd64 $(GO_ENV) $(GO_CMD) build -ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(BINARY_UNIX)_amd64 ./cmd/wemixvisor
	@GOOS=linux GOARCH=arm64 $(GO_ENV) $(GO_CMD) build -ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(BINARY_UNIX)_arm64 ./cmd/wemixvisor
	@echo "$(COLOR_GREEN)✓ Linux binaries built$(COLOR_RESET)"

build-darwin: ## Build binary for macOS
	@echo "$(COLOR_YELLOW)Building for macOS...$(COLOR_RESET)"
	@mkdir -p $(BIN_DIR)
	@GOOS=darwin GOARCH=amd64 $(GO_ENV) $(GO_CMD) build -ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(BINARY_DARWIN)_amd64 ./cmd/wemixvisor
	@GOOS=darwin GOARCH=arm64 $(GO_ENV) $(GO_CMD) build -ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(BINARY_DARWIN)_arm64 ./cmd/wemixvisor
	@echo "$(COLOR_GREEN)✓ macOS binaries built$(COLOR_RESET)"

build-windows: ## Build binary for Windows
	@echo "$(COLOR_YELLOW)Building for Windows...$(COLOR_RESET)"
	@mkdir -p $(BIN_DIR)
	@GOOS=windows GOARCH=amd64 $(GO_ENV) $(GO_CMD) build -ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(BINARY_WINDOWS) ./cmd/wemixvisor
	@echo "$(COLOR_GREEN)✓ Windows binary built$(COLOR_RESET)"

install: ## Install binary to $GOPATH/bin
	@echo "$(COLOR_GREEN)Installing $(BINARY_NAME)...$(COLOR_RESET)"
	@$(GO_ENV) $(GO_CMD) install -ldflags "$(LDFLAGS)" ./cmd/wemixvisor
	@echo "$(COLOR_GREEN)✓ Installed to $(shell $(GO_CMD) env GOPATH)/bin/$(BINARY_NAME)$(COLOR_RESET)"

##@ Testing

test: test-unit ## Run unit tests (default)

test-unit: ## Run unit tests with race detector
	@echo "$(COLOR_GREEN)Running unit tests...$(COLOR_RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GO_ENV) $(GO_CMD) test -v -race -short -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@echo "$(COLOR_GREEN)✓ Unit tests completed$(COLOR_RESET)"

test-integration: ## Run integration tests
	@echo "$(COLOR_YELLOW)Running integration tests...$(COLOR_RESET)"
	@mkdir -p $(REPORTS_DIR)
	@$(GO_ENV) $(GO_CMD) test -v -tags=integration ./... 2>&1 | tee $(REPORTS_DIR)/integration-tests.log
	@echo "$(COLOR_GREEN)✓ Integration tests completed$(COLOR_RESET)"

test-e2e: ## Run end-to-end tests
	@echo "$(COLOR_YELLOW)Running E2E tests...$(COLOR_RESET)"
	@mkdir -p $(REPORTS_DIR)
	@$(GO_ENV) $(GO_CMD) test -v -tags=e2e ./test/e2e/... 2>&1 | tee $(REPORTS_DIR)/e2e-tests.log
	@echo "$(COLOR_GREEN)✓ E2E tests completed$(COLOR_RESET)"

test-all: test-unit test-integration test-e2e ## Run all tests

test-verbose: ## Run tests with verbose output
	@echo "$(COLOR_GREEN)Running tests with verbose output...$(COLOR_RESET)"
	@mkdir -p $(REPORTS_DIR)
	@$(GO_ENV) $(GO_CMD) test -v -race ./... 2>&1 | tee $(REPORTS_DIR)/test-verbose.log

test-packages: ## Run tests for specific packages
	@echo "$(COLOR_GREEN)Testing core packages...$(COLOR_RESET)"
	@$(GO_ENV) $(GO_CMD) test -v ./internal/config/
	@$(GO_ENV) $(GO_CMD) test -v ./internal/metrics/
	@$(GO_ENV) $(GO_CMD) test -v ./internal/alerting/
	@$(GO_ENV) $(GO_CMD) test -v ./internal/api/
	@$(GO_ENV) $(GO_CMD) test -v ./internal/performance/
	@$(GO_ENV) $(GO_CMD) test -v ./pkg/types/
	@$(GO_ENV) $(GO_CMD) test -v ./pkg/logger/

##@ Coverage

coverage: ## Generate coverage report
	@echo "$(COLOR_GREEN)Generating coverage report...$(COLOR_RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GO_ENV) $(GO_CMD) test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@$(GO_ENV) $(GO_CMD) tool cover -func=$(COVERAGE_DIR)/coverage.out | tee $(COVERAGE_DIR)/coverage.txt
	@echo "$(COLOR_GREEN)✓ Coverage report: $(COVERAGE_DIR)/coverage.txt$(COLOR_RESET)"

coverage-html: coverage ## Generate HTML coverage report
	@echo "$(COLOR_GREEN)Generating HTML coverage report...$(COLOR_RESET)"
	@$(GO_ENV) $(GO_CMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(COLOR_GREEN)✓ HTML report: $(COVERAGE_DIR)/coverage.html$(COLOR_RESET)"

coverage-func: ## Show function-level coverage
	@mkdir -p $(COVERAGE_DIR)
	@$(GO_ENV) $(GO_CMD) test -coverprofile=$(COVERAGE_DIR)/coverage.out ./... > /dev/null 2>&1
	@$(GO_ENV) $(GO_CMD) tool cover -func=$(COVERAGE_DIR)/coverage.out

##@ Code Quality

fmt: ## Format code
	@echo "$(COLOR_GREEN)Formatting code...$(COLOR_RESET)"
	@$(GO_ENV) $(GO_CMD) fmt ./...
	@echo "$(COLOR_GREEN)✓ Code formatted$(COLOR_RESET)"

vet: ## Run go vet
	@echo "$(COLOR_GREEN)Running go vet...$(COLOR_RESET)"
	@$(GO_ENV) $(GO_CMD) vet ./...
	@echo "$(COLOR_GREEN)✓ Vet completed$(COLOR_RESET)"

lint: ## Run linter
	@echo "$(COLOR_GREEN)Running linter...$(COLOR_RESET)"
	@if command -v golangci-lint > /dev/null; then \
		$(GO_ENV) golangci-lint run ./...; \
		echo "$(COLOR_GREEN)✓ Lint completed$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ golangci-lint not found, skipping...$(COLOR_RESET)"; \
	fi

check: fmt vet lint ## Run all code quality checks

##@ Profiling

profile-cpu: ## Run CPU profiling
	@echo "$(COLOR_YELLOW)Running CPU profiling...$(COLOR_RESET)"
	@mkdir -p $(PROFILES_DIR)
	@$(GO_ENV) $(GO_CMD) test -cpuprofile=$(PROFILES_DIR)/cpu.prof -bench=. ./...
	@echo "$(COLOR_GREEN)✓ CPU profile: $(PROFILES_DIR)/cpu.prof$(COLOR_RESET)"

profile-mem: ## Run memory profiling
	@echo "$(COLOR_YELLOW)Running memory profiling...$(COLOR_RESET)"
	@mkdir -p $(PROFILES_DIR)
	@$(GO_ENV) $(GO_CMD) test -memprofile=$(PROFILES_DIR)/mem.prof -bench=. ./...
	@echo "$(COLOR_GREEN)✓ Memory profile: $(PROFILES_DIR)/mem.prof$(COLOR_RESET)"

##@ Docker

docker: docker-build ## Build Docker image (alias)

docker-build: ## Build Docker image
	@echo "$(COLOR_GREEN)Building Docker image...$(COLOR_RESET)"
	@docker build -t wemixvisor:$(VERSION) -t wemixvisor:latest .
	@echo "$(COLOR_GREEN)✓ Docker image built: wemixvisor:$(VERSION)$(COLOR_RESET)"

docker-push: ## Push Docker image to registry
	@echo "$(COLOR_YELLOW)Pushing Docker image...$(COLOR_RESET)"
	@docker push wemixvisor:$(VERSION)
	@docker push wemixvisor:latest
	@echo "$(COLOR_GREEN)✓ Docker image pushed$(COLOR_RESET)"

##@ Release

release: clean check test build-all ## Prepare release (clean, check, test, build all)
	@echo "$(COLOR_GREEN)Creating release artifacts...$(COLOR_RESET)"
	@mkdir -p $(DIST_DIR)/releases
	@cd $(BIN_DIR) && tar czf ../releases/$(BINARY_NAME)_$(VERSION)_linux_amd64.tar.gz $(BINARY_UNIX)_amd64
	@cd $(BIN_DIR) && tar czf ../releases/$(BINARY_NAME)_$(VERSION)_linux_arm64.tar.gz $(BINARY_UNIX)_arm64
	@cd $(BIN_DIR) && tar czf ../releases/$(BINARY_NAME)_$(VERSION)_darwin_amd64.tar.gz $(BINARY_DARWIN)_amd64
	@cd $(BIN_DIR) && tar czf ../releases/$(BINARY_NAME)_$(VERSION)_darwin_arm64.tar.gz $(BINARY_DARWIN)_arm64
	@cd $(BIN_DIR) && zip -q ../releases/$(BINARY_NAME)_$(VERSION)_windows_amd64.zip $(BINARY_WINDOWS)
	@shasum -a 256 $(DIST_DIR)/releases/* > $(DIST_DIR)/releases/checksums.txt
	@echo "$(COLOR_GREEN)✓ Release artifacts created in $(DIST_DIR)/releases/$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Release $(VERSION) ready!$(COLOR_RESET)"
	@ls -lh $(DIST_DIR)/releases/

release-notes: ## Generate release notes
	@echo "$(COLOR_GREEN)Generating release notes...$(COLOR_RESET)"
	@git log $(shell git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~10")..HEAD --pretty=format:"- %s" > $(DIST_DIR)/RELEASE_NOTES.md
	@echo "$(COLOR_GREEN)✓ Release notes: $(DIST_DIR)/RELEASE_NOTES.md$(COLOR_RESET)"

##@ Cleanup

clean: ## Remove build artifacts
	@echo "$(COLOR_YELLOW)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(DIST_DIR)
	@rm -rf bin/
	@rm -f $(BINARY_NAME)
	@rm -f *.test
	@rm -f *.out
	@rm -f coverage*.out
	@rm -f cli_coverage.out
	@echo "$(COLOR_GREEN)✓ Clean completed$(COLOR_RESET)"

clean-all: clean ## Remove all generated files including caches
	@echo "$(COLOR_YELLOW)Deep cleaning...$(COLOR_RESET)"
	@rm -rf .cache .gomodcache
	@$(GO_ENV) $(GO_CMD) clean -cache -testcache -modcache
	@echo "$(COLOR_GREEN)✓ Deep clean completed$(COLOR_RESET)"

clean-test: ## Remove test artifacts
	@echo "$(COLOR_YELLOW)Cleaning test artifacts...$(COLOR_RESET)"
	@find . -name "*.test" -type f -delete
	@find . -name "*.out" -type f -delete
	@rm -rf $(COVERAGE_DIR) $(REPORTS_DIR)
	@echo "$(COLOR_GREEN)✓ Test artifacts cleaned$(COLOR_RESET)"

##@ Development

dev: ## Build and run in development mode
	@echo "$(COLOR_GREEN)Building for development...$(COLOR_RESET)"
	@$(MAKE) build
	@echo "$(COLOR_BLUE)Running $(BINARY_NAME)...$(COLOR_RESET)"
	@$(BIN_DIR)/$(BINARY_NAME) --help

watch: ## Watch for changes and rebuild (requires entr)
	@if command -v entr > /dev/null; then \
		echo "$(COLOR_BLUE)Watching for changes...$(COLOR_RESET)"; \
		find . -name "*.go" | entr -r make build; \
	else \
		echo "$(COLOR_YELLOW)⚠ entr not found. Install with: brew install entr$(COLOR_RESET)"; \
	fi

deps: ## Download dependencies
	@echo "$(COLOR_GREEN)Downloading dependencies...$(COLOR_RESET)"
	@$(GO_ENV) $(GO_CMD) mod download
	@$(GO_ENV) $(GO_CMD) mod verify
	@echo "$(COLOR_GREEN)✓ Dependencies downloaded$(COLOR_RESET)"

deps-update: ## Update dependencies
	@echo "$(COLOR_YELLOW)Updating dependencies...$(COLOR_RESET)"
	@$(GO_ENV) $(GO_CMD) get -u ./...
	@$(GO_ENV) $(GO_CMD) mod tidy
	@echo "$(COLOR_GREEN)✓ Dependencies updated$(COLOR_RESET)"

deps-tidy: ## Tidy dependencies
	@echo "$(COLOR_GREEN)Tidying dependencies...$(COLOR_RESET)"
	@$(GO_ENV) $(GO_CMD) mod tidy
	@echo "$(COLOR_GREEN)✓ Dependencies tidied$(COLOR_RESET)"

##@ Scripts

scripts-run-tests: ## Run test scripts
	@echo "$(COLOR_GREEN)Running test scripts...$(COLOR_RESET)"
	@mkdir -p $(SCRIPTS_OUTPUT_DIR)
	@bash scripts/run_tests.sh 2>&1 | tee $(SCRIPTS_OUTPUT_DIR)/run_tests.log
	@echo "$(COLOR_GREEN)✓ Test scripts completed$(COLOR_RESET)"

scripts-setup-test: ## Setup test environment
	@echo "$(COLOR_GREEN)Setting up test environment...$(COLOR_RESET)"
	@mkdir -p $(SCRIPTS_OUTPUT_DIR)
	@bash scripts/setup_test_env.sh ~/.wemixd_test 2>&1 | tee $(SCRIPTS_OUTPUT_DIR)/setup_test.log
	@echo "$(COLOR_GREEN)✓ Test environment setup completed$(COLOR_RESET)"

scripts-list: ## List available scripts
	@echo "$(COLOR_BOLD)Available Scripts:$(COLOR_RESET)"
	@ls -1 scripts/*.sh | sed 's|scripts/||' | sed 's|\.sh||'

##@ Information

version: ## Show version information
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(COMMIT)"
	@echo "Git Tag:    $(GIT_TAG)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(shell $(GO_CMD) version)"

info: ## Show build information
	@echo "$(COLOR_BOLD)Build Configuration$(COLOR_RESET)"
	@echo "  Version:     $(VERSION)"
	@echo "  Commit:      $(COMMIT)"
	@echo "  Tag:         $(GIT_TAG)"
	@echo "  Build Date:  $(BUILD_DATE)"
	@echo "  Go Command:  $(GO_CMD)"
	@echo "  Go Version:  $(shell $(GO_CMD) version | cut -d' ' -f3-)"
	@echo ""
	@echo "$(COLOR_BOLD)Directories$(COLOR_RESET)"
	@echo "  Dist:          $(DIST_DIR)"
	@echo "  Binary:        $(BIN_DIR)"
	@echo "  Coverage:      $(COVERAGE_DIR)"
	@echo "  Reports:       $(REPORTS_DIR)"
	@echo "  Profiles:      $(PROFILES_DIR)"
	@echo "  Scripts Out:   $(SCRIPTS_OUTPUT_DIR)"

tree: ## Show project directory tree
	@if command -v tree > /dev/null; then \
		tree -L 3 -I 'vendor|.git|.gomodcache|.cache|dist|node_modules'; \
	else \
		find . -maxdepth 3 -type d -not -path '*/\.*' -not -path '*/vendor/*' | sed 's|[^/]*/| |g'; \
	fi
