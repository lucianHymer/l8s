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

# Build tags to exclude problematic storage drivers
BUILD_TAGS := exclude_graphdriver_btrfs,exclude_graphdriver_devicemapper

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
	$(GOBUILD) $(GOFLAGS) -tags $(BUILD_TAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR)
	$(GOCLEAN)
	@echo "✓ Clean complete"

.PHONY: test
test: test-go test-zsh ## Run all tests (Go unit tests and ZSH plugin tests)

.PHONY: test-go
test-go: ## Run Go unit tests
	@echo "🧪 Running Go unit tests..."
	@$(GOTEST) -race -tags test,$(BUILD_TAGS) -coverprofile=coverage.out ./pkg/... ./cmd/...
	@echo "✓ Go unit tests complete"

.PHONY: test-zsh
test-zsh: ## Run ZSH plugin tests
	@echo "🐚 Running ZSH plugin tests..."
	@cd host-integration/oh-my-zsh/l8s/tests && zsh run_all_tests.sh
	@echo "✓ ZSH plugin tests complete"

.PHONY: test-coverage
test-coverage: test-go ## Run tests with coverage report
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
test-all: test test-integration ## Run all tests (unit, ZSH, and integration)

.PHONY: lint
lint: check-deps ## Run linters
	@echo "🔍 Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --build-tags $(BUILD_TAGS) ./...; \
	else \
		echo "⚠️  golangci-lint not installed"; \
		echo "Skipping lint step. To enable linting, install golangci-lint:"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
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

.PHONY: install-hooks
install-hooks: ## Install git hooks for local CI
	@echo "🪝 Installing git hooks..."
	@echo '#!/bin/sh' > .git/hooks/pre-push
	@echo 'echo "Running CI checks before push..."' >> .git/hooks/pre-push
	@echo 'make ci' >> .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "✓ Git hooks installed! Tests will run before each push."

.PHONY: zsh-plugin
zsh-plugin: ## Install l8s ZSH completion plugin on host machine
	@echo "🐚 Installing l8s ZSH completion plugin..."
	@if [ ! -d "$$HOME/.oh-my-zsh" ]; then \
		echo "❌ Oh My Zsh not found. Please install Oh My Zsh first."; \
		exit 1; \
	fi
	@echo "📁 Copying plugin to Oh My Zsh custom plugins..."
	@mkdir -p $$HOME/.oh-my-zsh/custom/plugins
	@cp -r pkg/embed/dotfiles/.oh-my-zsh/custom/plugins/l8s $$HOME/.oh-my-zsh/custom/plugins/
	@echo "✓ Plugin copied"
	@echo "📝 Updating .zshrc..."
	@if ! grep -q "# l8s plugin auto-load" $$HOME/.zshrc; then \
		echo "" >> $$HOME/.zshrc; \
		echo "# l8s plugin auto-load" >> $$HOME/.zshrc; \
		echo 'if [[ -d "$$ZSH_CUSTOM/plugins/l8s" ]]; then' >> $$HOME/.zshrc; \
		echo '    plugins+=(l8s)' >> $$HOME/.zshrc; \
		echo 'fi' >> $$HOME/.zshrc; \
		echo "✓ Added l8s plugin to .zshrc"; \
	else \
		echo "✓ l8s plugin already configured in .zshrc"; \
	fi
	@echo "🎉 Installation complete! Restart your shell or run: source ~/.zshrc"

.PHONY: container-build
container-build: ## Build the l8s container image
	@echo "🐳 Building container image..."
	@if [ ! -f containers/Containerfile ]; then \
		echo "❌ Error: containers/Containerfile not found"; \
		exit 1; \
	fi
	podman build \
		--build-arg CACHEBUST=$$(date +%s) \
		-t localhost/l8s-fedora:latest -f containers/Containerfile containers/
	@echo "✓ Container image built"

.PHONY: container-build-test
container-build-test: ## Build the test container image
	@echo "🧪 Building test container image..."
	@if [ ! -f containers/Containerfile.test ]; then \
		echo "❌ Error: containers/Containerfile.test not found"; \
		exit 1; \
	fi
	podman build \
		--build-arg CACHEBUST=$$(date +%s) \
		-t localhost/l8s-fedora:test -f containers/Containerfile.test containers/
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
	@echo "✓ Go is installed"
	@echo "ℹ️  Using build tags to exclude optional dependencies: $(BUILD_TAGS)"

.PHONY: setup
setup: check-deps deps check-podman ## Initial project setup
	@echo "🔧 Setting up development environment..."
	@echo "✓ Setup complete"

.PHONY: update-nvim
update-nvim: ## Update Neovim plugins in pkg/embed/dotfiles
	@echo "📦 Updating Neovim plugins in pkg/embed/dotfiles/.config/nvim..."
	@cd pkg/embed/dotfiles/.config/nvim && \
		HOME=$(PWD)/pkg/embed/dotfiles nvim --headless "+autocmd User LazySync quitall" "+Lazy sync" 2>&1
	@if [ $$? -eq 0 ]; then \
		echo "✓ Plugin update completed successfully"; \
		if git diff --quiet lazy-lock.json 2>/dev/null; then \
			echo "ℹ️  No plugin updates available"; \
		else \
			echo "📝 lazy-lock.json has been updated"; \
			echo "⚠️  Remember to commit the changes if any plugins were updated"; \
		fi \
	else \
		echo "❌ Plugin update failed"; \
		exit 1; \
	fi

# CI/CD targets
.PHONY: ci
ci: clean deps lint test update-nvim ## Run CI pipeline

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
