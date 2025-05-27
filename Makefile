# Secure Filesystem MCP Server Makefile

# Binary name and paths
BINARY_NAME=filesystem
BUILD_DIR=bin
CMD_DIR=cmd/filesystem

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags="-s -w"
BUILDFLAGS=-v

# Default target
.PHONY: all
all: clean deps build

# Clean build artifacts
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Download dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build the binary
.PHONY: build
build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

# Build optimized production binary
.PHONY: build-prod
build-prod:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GOTEST) -v -cover ./...

# Run tests with race detection
.PHONY: test-race
test-race:
	$(GOTEST) -v -race ./...

# Run benchmarks
.PHONY: bench
bench:
	$(GOTEST) -bench=. -benchmem ./...

# Format code
.PHONY: fmt
fmt:
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	golangci-lint run

# Security scan (requires gosec)
.PHONY: security
security:
	gosec ./...

# Generate code documentation
.PHONY: docs
docs:
	$(GOCMD) doc -all ./...

# Install the binary to GOPATH/bin
.PHONY: install
install:
	$(GOCMD) install ./$(CMD_DIR)

# Cross-compile for multiple platforms
.PHONY: build-all
build-all: clean deps
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)

# Development setup - install all tools
.PHONY: dev-setup
dev-setup:
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	$(GOGET) -u github.com/securecodewarrior/gosec/v2/cmd/gosec

# Run the server with example configuration
.PHONY: run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) .

# Run the server with config file
.PHONY: run-config
run-config: build
	./$(BUILD_DIR)/$(BINARY_NAME) -config config.yaml

# Quick development cycle
.PHONY: dev
dev: fmt lint test build

# CI/CD pipeline targets
.PHONY: ci
ci: deps fmt lint security test-race build-prod

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all         - Clean, download deps, and build"
	@echo "  clean       - Remove build artifacts"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  build       - Build development binary"
	@echo "  build-prod  - Build optimized production binary"
	@echo "  build-all   - Cross-compile for all platforms"
	@echo "  test        - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  test-race   - Run tests with race detection"
	@echo "  bench       - Run benchmarks"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code (requires golangci-lint)"
	@echo "  security    - Security scan (requires gosec)"
	@echo "  docs        - Generate documentation"
	@echo "  install     - Install binary to GOPATH/bin"
	@echo "  run         - Run server with current directory"
	@echo "  run-config  - Run server with config.yaml"
	@echo "  dev         - Quick development cycle (fmt, lint, test, build)"
	@echo "  dev-setup   - Install development tools"
	@echo "  ci          - CI/CD pipeline"
	@echo "  help        - Show this help" 