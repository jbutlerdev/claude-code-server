# Makefile for claude-code-server MCP server

# Binary name
BINARY_NAME=claude-code-mcp-server

# Build directory
BUILD_DIR=../../build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOMOD=$(GOCMD) mod
GOCLEAN=$(GOCMD) clean

# Version and build information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker flags for static compilation and version info
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME) -w -s -extldflags '-static'"

# Build flags for static compilation
BUILD_FLAGS=-a -installsuffix cgo

.PHONY: all build build-static clean deps test help

all: build

## Build the binary (dynamically linked)
build:
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) .

## Build statically linked binary (recommended for distribution)
build-static:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

## Build statically linked binary for current platform
build-current-static:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

## Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## Run tests
test:
	$(GOCMD) test -v ./...

## Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

## Display help
help:
	@echo "Available targets:"
	@echo "  build              - Build the binary (dynamically linked)"
	@echo "  build-static       - Build statically linked binaries for multiple platforms"
	@echo "  build-current-static - Build statically linked binary for current platform"
	@echo "  deps               - Download and tidy dependencies"
	@echo "  test               - Run tests"
	@echo "  clean              - Clean build artifacts"
	@echo "  help               - Display this help"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION            - Version string (default: git describe)"
	@echo "  COMMIT             - Git commit hash (default: git rev-parse HEAD)"
	@echo "  BUILD_TIME         - Build timestamp (default: current UTC time)"