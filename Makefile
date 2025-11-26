###################################################
###             Lumera SDK-Go Makefile          ###
###################################################
# Go SDK + examples for the Lumera blockchain.
# Run `make` (or `make sdk` / `make examples`) to build; `make help` lists targets.

.PHONY: help all sdk build examples example-% test lint clean tidy deps install install-tools

GO ?= go
GOLANGCI_LINT ?= golangci-lint
BUILD_DIR ?= build
EXAMPLES ?= action-approve cascade-upload cascade-download query-actions claim-tokens ica-request-tx ica-approve-tx

# Default target: build SDK and examples
all: sdk examples ## Build SDK (compile packages) and all examples

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z0-9_/-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-28s\033[0m %s\n", $$1, $$2}'

# SDK/library compile check (Go libs are compiled as part of go build; no artifact)
sdk: go.sum ## Compile all packages in the module (verifies SDK builds)
	@echo "Compiling SDK (library packages)..."
	@$(GO) build ./...
	@echo "SDK build completed successfully."

# Alias for backward compatibility
build: sdk ## Alias for sdk

go.sum: go.mod
	@echo "Verifying and tidying go modules..."
	@$(GO) mod verify
	@$(GO) mod tidy

# Examples to build (main packages under ./examples)
examples: $(EXAMPLES:%=example-%) ## Build all example binaries into ./build
	@echo "Examples built into $(BUILD_DIR)/"

# Build a single example: make example-cascade-upload
example-%: ## Build a single example binary into ./build (usage: make example-cascade-upload)
	@mkdir -p $(BUILD_DIR)
	@echo "Building $*..."
	@$(GO) build -o $(BUILD_DIR)/$* ./examples/$*

test: ## Run tests with race detector and coverage
	@echo "Running tests..."
	@$(GO) test -v -race -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html

lint: ## Run linters (requires golangci-lint)
	@echo "Running linters..."
	@$(GOLANGCI_LINT) run ./... --timeout=5m

install-tools: ## Install required developer tools
	@echo "Installing golangci-lint (latest)..."
	@$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tool installation complete."

tidy: ## Tidy go modules
	@$(GO) mod tidy

deps: ## Update dependencies (minor/patch) then tidy
	@$(GO) get -u ./...
	@$(GO) mod tidy

install: ## Install all module binaries (none if project has only libraries and examples)
	@echo "Installing binaries (if any main packages outside examples)..."
	@$(GO) install ./...

clean: ## Clean build artifacts and test caches
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
	@$(GO) clean -cache -testcache
