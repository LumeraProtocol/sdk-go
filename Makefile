.PHONY: help all sdk build examples example-% test lint clean tidy deps install

# Default target: build SDK and examples
all: sdk examples ## Build SDK (compile packages) and all examples

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z0-9_/-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-28s\033[0m %s\n", $$1, $$2}'

# SDK/library compile check (Go libs are compiled as part of go build; no artifact)
sdk: ## Compile all packages in the module (verifies SDK builds)
	@echo "Compiling SDK (library packages)..."
	@go build ./...

# Alias for backward compatibility
build: sdk ## Alias for sdk

# Examples to build (main packages under ./examples)
EXAMPLES := action-approve cascade-upload cascade-download query-actions claim-tokens ica-request-tx ica-approve-tx
BUILD_DIR := build

examples: $(EXAMPLES:%=example-%) ## Build all example binaries into ./build
	@echo "Examples built into $(BUILD_DIR)/"

# Build a single example: make example-cascade-upload
example-%: ## Build a single example binary into ./build (usage: make example-cascade-upload)
	@mkdir -p $(BUILD_DIR)
	@echo "Building $*..."
	@go build -o $(BUILD_DIR)/$* ./examples/$*

test: ## Run tests with race detector and coverage
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

lint: ## Run linters (requires golangci-lint)
	@echo "Running linters..."
	@golangci-lint run --timeout=5m

tidy: ## Tidy go modules
	@go mod tidy

deps: ## Update dependencies (minor/patch) then tidy
	@go get -u ./...
	@go mod tidy

install: ## Install all module binaries (none if project has only libraries and examples)
	@echo "Installing binaries (if any main packages outside examples)..."
	@go install ./...

clean: ## Clean build artifacts and test caches
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
	@go clean -cache -testcache