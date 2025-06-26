# Claude Code Server Makefile

.PHONY: help build run test clean dev docker-build docker-run deps lint fmt migrate-up migrate-down generate

# Default target
help: ## Show this help message
	@echo 'Usage: make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build configuration
BINARY_NAME=claude-code-server
BUILD_DIR=bin
DOCKER_IMAGE=claude-code-server
DOCKER_TAG=latest

# Go configuration
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build the application
build: ## Build the application binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application locally
run: build ## Build and run the application
	@echo "Starting $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Development server with auto-reload
dev: ## Start development server with air (auto-reload)
	@if ! command -v air > /dev/null; then \
		echo "Installing air for hot reload..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	air

# Install dependencies
deps: ## Download and install dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@if ! command -v claude > /dev/null; then \
		echo "Installing Claude Code CLI..."; \
		npm install -g @anthropic-ai/claude-code; \
	fi

# Run tests
test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Tests complete. Coverage report: coverage.html"

# Run tests with short flag
test-short: ## Run tests with short flag
	@echo "Running short tests..."
	$(GOTEST) -short -v ./...

# Run integration tests
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GOTEST) -tags=integration -v ./tests/...

# Lint the code
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

# Format the code
fmt: ## Format Go code
	@echo "Formatting code..."
	$(GOFMT) ./...
	goimports -w .

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Database migrations
migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "ERROR: DATABASE_URL environment variable is required"; \
		exit 1; \
	fi
	migrate -path internal/database/migrations -database "$(DATABASE_URL)" up

migrate-down: ## Run database migrations down
	@echo "Running migrations down..."
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "ERROR: DATABASE_URL environment variable is required"; \
		exit 1; \
	fi
	migrate -path internal/database/migrations -database "$(DATABASE_URL)" down

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "ERROR: NAME is required. Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	migrate create -ext sql -dir internal/database/migrations -seq $(NAME)

# Docker commands
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## Run application in Docker
	@echo "Running Docker container..."
	docker run -d \
		--name $(BINARY_NAME) \
		-p 8080:8080 \
		-e DATABASE_URL="$(DATABASE_URL)" \
		-e CLAUDE_CODE_API_KEY="$(CLAUDE_CODE_API_KEY)" \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-stop: ## Stop Docker container
	docker stop $(BINARY_NAME) || true
	docker rm $(BINARY_NAME) || true

docker-compose-up: ## Start services with docker-compose
	@echo "Starting services with docker-compose..."
	docker-compose up -d

docker-compose-down: ## Stop services with docker-compose
	@echo "Stopping services with docker-compose..."
	docker-compose down

docker-compose-logs: ## View docker-compose logs
	docker-compose logs -f

# Generate OpenAPI types
generate: ## Generate OpenAPI types and documentation
	@echo "Generating OpenAPI types..."
	@if ! command -v oapi-codegen > /dev/null; then \
		echo "Installing oapi-codegen..."; \
		go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest; \
	fi
	# Generate types from OpenAPI spec
	# oapi-codegen -generate types -package api api/openapi.yaml > internal/api/types.go

# Security scan
security: ## Run security scan
	@echo "Running security scan..."
	@if ! command -v gosec > /dev/null; then \
		echo "Installing gosec..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	gosec ./...

# Check for outdated dependencies
deps-check: ## Check for outdated dependencies
	@echo "Checking for outdated dependencies..."
	go list -u -m all

# Update dependencies
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

# Create .env file from template
env: ## Create .env file from template
	@if [ ! -f .env ]; then \
		echo "Creating .env file..."; \
		cp .env.example .env; \
		echo "Please edit .env file with your configuration"; \
	else \
		echo ".env file already exists"; \
	fi

# Setup development environment
setup: deps env ## Setup development environment
	@echo "Setting up development environment..."
	@echo "1. Install PostgreSQL and create database"
	@echo "2. Set CLAUDE_CODE_API_KEY in .env file"
	@echo "3. Set DATABASE_URL in .env file"
	@echo "4. Run 'make migrate-up' to setup database"
	@echo "5. Run 'make dev' to start development server"

# Production build
build-prod: ## Build production binary
	@echo "Building production binary..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) \
		-ldflags="-w -s -X main.version=$(shell git describe --tags --abbrev=0 2>/dev/null || echo 'dev')" \
		-o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

# Release build (all platforms)
build-release: ## Build release binaries for all platforms
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/server

# Benchmark tests
bench: ## Run benchmark tests
	@echo "Running benchmark tests..."
	$(GOTEST) -bench=. -benchmem ./...

# Load test
load-test: ## Run load tests (requires k6)
	@if ! command -v k6 > /dev/null; then \
		echo "k6 is required for load testing. Install from https://k6.io/"; \
		exit 1; \
	fi
	k6 run tests/load/basic.js