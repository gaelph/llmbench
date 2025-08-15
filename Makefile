# LLMBench Makefile

# Variables
BINARY_NAME=llmbench
VERSION?=dev
BUILD_DIR=build
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME) .

# Build for multiple platforms
.PHONY: build-all
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Built binaries in $(BUILD_DIR)/"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Lint the code
.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run

# Format the code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Vet the code
.PHONY: vet
vet:
	@echo "Vetting code..."
	@go vet ./...

# Run all checks (format, vet, lint, test)
.PHONY: check
check: fmt vet lint test

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Install the binary
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install .

# Uninstall the binary
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(shell go env GOPATH)/bin/$(BINARY_NAME)

# Run the application with sample config
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME)

# Initialize sample configuration
.PHONY: init-config
init-config: build
	@echo "Initializing sample configuration..."
	@./$(BINARY_NAME) config init

# Test connections with sample config
.PHONY: test-connections
test-connections: build
	@echo "Testing connections..."
	@./$(BINARY_NAME) test

# Run a sample benchmark
.PHONY: run-benchmark
run-benchmark: build
	@echo "Running sample benchmark..."
	@./$(BINARY_NAME) benchmark -m "Hello, how are you?" -r 5

# Run interactive mode
.PHONY: run-interactive
run-interactive: build
	@echo "Running in interactive mode..."
	@./$(BINARY_NAME) benchmark --interactive

# Development setup
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go mod download
	@go mod tidy
	@echo "Development environment ready!"

# Create release
.PHONY: release
release: clean test build-all
	@echo "Creating release $(VERSION)..."
	@cd $(BUILD_DIR) && \
	for binary in *; do \
		if [[ $$binary == *.exe ]]; then \
			zip $${binary%.exe}.zip $$binary; \
		else \
			tar -czf $$binary.tar.gz $$binary; \
		fi; \
	done
	@echo "Release $(VERSION) created in $(BUILD_DIR)/"

# Docker build
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):$(VERSION) .

# Docker run
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	@docker run --rm -it $(BINARY_NAME):$(VERSION)

# Show help
.PHONY: help
help:
	@echo "LLMBench Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build           Build the binary"
	@echo "  build-all       Build for multiple platforms"
	@echo "  test            Run tests"
	@echo "  test-coverage   Run tests with coverage report"
	@echo "  bench           Run benchmarks"
	@echo "  lint            Run linter"
	@echo "  fmt             Format code"
	@echo "  vet             Vet code"
	@echo "  check           Run all checks (fmt, vet, lint, test)"
	@echo "  clean           Clean build artifacts"
	@echo "  deps            Install dependencies"
	@echo "  install         Install binary to GOPATH/bin"
	@echo "  uninstall       Remove binary from GOPATH/bin"
	@echo "  run             Run the application"
	@echo "  init-config     Initialize sample configuration"
	@echo "  test-connections Test connections with providers"
	@echo "  run-benchmark   Run a sample benchmark"
	@echo "  run-interactive Run in interactive mode"
	@echo "  dev-setup       Setup development environment"
	@echo "  release         Create release builds"
	@echo "  docker-build    Build Docker image"
	@echo "  docker-run      Run Docker container"
	@echo "  help            Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION         Version to build (default: dev)"
	@echo ""
	@echo "Examples:"
	@echo "  make build VERSION=v1.0.0"
	@echo "  make release VERSION=v1.0.0"
	@echo "  make test-coverage"
