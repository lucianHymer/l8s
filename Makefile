# l8s Makefile

# Variables
BINARY_NAME := l8s
BUILD_DIR := build
INSTALL_DIR := /usr/local/bin
GO := go
GOFLAGS := -v
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOCLEAN := $(GO) clean
GOMOD := $(GO) mod
MAIN_PACKAGE := ./cmd/l8s

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Detect OS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    OS := linux
endif
ifeq ($(UNAME_S),Darwin)
    OS := darwin
endif

# Default target
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo "l8s - The container management system that really ties the room together"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: all
all: clean test build ## Clean, test, and build

.PHONY: build
build: check-deps ## Build the l8s binary
	@echo "🎳 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(GOFLAGS) -tags exclude_graphdriver_btrfs,exclude_graphdriver_devicemapper $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR)
	$(GOCLEAN)
	@echo "✓ Clean complete"

.PHONY: test
test: ## Run unit tests
	@echo "🧪 Running unit tests..."
	$(GOTEST) -v -race -tags test -coverprofile=coverage.out ./pkg/... ./cmd/...
	@echo "✓ Unit tests complete"

.PHONY: test-coverage
test-coverage: test ## Run tests with coverage report
	@echo "📊 Generating coverage report..."
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

.PHONY: test-integration
test-integration: ## Run integration tests (requires Podman)
	@echo "🔧 Running integration tests..."
	@if ! command -v podman >/dev/null 2>&1; then \
		echo "❌ Error: Podman is required for integration tests"; \
		exit 1; \
	fi
	$(GOTEST) -v -tags=integration -timeout=10m ./test/integration/...
	@echo "✓ Integration tests complete"

.PHONY: test-all
test-all: test test-integration ## Run all tests

.PHONY: lint
lint: ## Run linters
	@echo "🔍 Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint not installed, using go vet"; \
		$(GO) vet ./...; \
	fi
	@echo "✓ Linting complete"

.PHONY: fmt
fmt: ## Format code
	@echo "✨ Formatting code..."
	@$(GO) fmt ./...
	@echo "✓ Formatting complete"

.PHONY: tidy
tidy: ## Tidy go modules
	@echo "📦 Tidying modules..."
	$(GOMOD) tidy
	@echo "✓ Module tidy complete"

.PHONY: deps
deps: ## Download dependencies
	@echo "📥 Downloading dependencies..."
	$(GOMOD) download
	@echo "✓ Dependencies downloaded"

.PHONY: install
install: build ## Install l8s to system
	@echo "📦 Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ Installation complete"

.PHONY: uninstall
uninstall: ## Uninstall l8s from system
	@echo "🗑️  Uninstalling $(BINARY_NAME)..."
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ Uninstallation complete"

.PHONY: container-build
container-build: ## Build the l8s container image
	@echo "🐳 Building container image..."
	@if [ ! -f containers/Containerfile ]; then \
		echo "❌ Error: containers/Containerfile not found"; \
		exit 1; \
	fi
	podman build -t localhost/l8s-fedora:latest -f containers/Containerfile containers/
	@echo "✓ Container image built"

.PHONY: container-build-test
container-build-test: ## Build the test container image
	@echo "🧪 Building test container image..."
	@if [ ! -f containers/Containerfile.test ]; then \
		echo "❌ Error: containers/Containerfile.test not found"; \
		exit 1; \
	fi
	podman build -t localhost/l8s-fedora:test -f containers/Containerfile.test containers/
	@echo "✓ Test container image built"

.PHONY: dev
dev: ## Run l8s in development mode
	@echo "🚀 Running in development mode..."
	$(GO) run $(MAIN_PACKAGE)

.PHONY: release
release: clean test lint build ## Build release binary
	@echo "📦 Building release binary..."
	GOOS=$(OS) GOARCH=amd64 $(GOBUILD) -v -a -installsuffix cgo $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$(OS)-amd64 $(MAIN_PACKAGE)
	@echo "✓ Release build complete: $(BUILD_DIR)/$(BINARY_NAME)-$(OS)-amd64"

.PHONY: check-podman
check-podman: ## Check if Podman is installed
	@if command -v podman >/dev/null 2>&1; then \
		echo "✓ Podman is installed: $$(podman --version)"; \
	else \
		echo "❌ Podman is not installed"; \
		echo "Please install Podman: https://podman.io/getting-started/installation"; \
		exit 1; \
	fi

.PHONY: check-deps
check-deps: ## Check build dependencies
	@echo "🔍 Checking build dependencies..."
	@if ! command -v $(GO) >/dev/null 2>&1; then \
		echo "❌ Go is not installed"; \
		echo "Please install Go 1.21+: https://go.dev/dl/"; \
		exit 1; \
	fi
	@if ! pkg-config --exists gpgme 2>/dev/null; then \
		echo "❌ gpgme is not installed"; \
		echo "Please install gpgme development package:"; \
		echo "  Fedora/RHEL: sudo dnf install -y gpgme-devel"; \
		echo "  Ubuntu/Debian: sudo apt-get install -y libgpgme-dev"; \
		echo "  macOS: brew install gpgme"; \
		exit 1; \
	fi
	@echo "✓ All build dependencies are installed"

.PHONY: setup
setup: check-deps deps check-podman ## Initial project setup
	@echo "🔧 Setting up development environment..."
	@echo "✓ Setup complete"

# CI/CD targets
.PHONY: ci
ci: clean deps lint test ## Run CI pipeline

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "⚡ Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...
	@echo "✓ Benchmarks complete"

# Watch for changes and run tests
.PHONY: watch
watch: ## Watch for changes and run tests
	@echo "👀 Watching for changes..."
	@if command -v entr >/dev/null 2>&1; then \
		find . -name '*.go' | entr -c make test; \
	else \
		echo "❌ entr is not installed. Install it to use watch mode."; \
		exit 1; \
	fi