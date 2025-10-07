.PHONY: help build test lint clean examples install

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the SDK
	@echo "Building SDK..."
	@go build ./...

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

lint: ## Run linters
	@echo "Running linters..."
	@golangci-lint run --timeout=5m

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f coverage.out coverage.html
	@go clean -cache -testcache

examples: ## Build all examples
	@echo "Building examples..."
	@cd examples/cascade-upload && go build
	@cd examples/cascade-download && go build
	@cd examples/query-actions && go build
	@cd examples/claim-tokens && go build

install: ## Install the SDK locally
	@echo "Installing SDK..."
	@go install ./...

tidy: ## Tidy go modules
	@go mod tidy

deps: ## Update dependencies
	@go get -u ./...
	@go mod tidy

