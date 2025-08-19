# Model Metadata Collection Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
BINARY_NAME=model-extractor
BINARY_UNIX=$(BINARY_NAME)_unix

# Build parameters
BUILD_DIR=build
MAIN_PATH=./cmd/model-extractor

# Default data paths
MODELS_INDEX_PATH=data/models-index.yaml
CATALOG_OUTPUT_PATH=data/models-catalog.yaml

.PHONY: all build clean test test-coverage lint fmt vet deps check help run process

# Default target
all: check build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Build for linux
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_UNIX) $(MAIN_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf output/
	rm -f data/hugging-face-redhat-ai-validated-*.yaml
	rm -f data/models-catalog.yaml

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. ./...

# Lint the code
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format the code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Vet the code
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

# Check code formatting
fmt-check:
	@echo "Checking code formatting..."
	@files=$$($(GOFMT) -l .); \
	if [ -n "$$files" ]; then \
		echo "The following files need formatting:"; \
		echo "$$files"; \
		exit 1; \
	fi

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run all checks
check: fmt-check vet lint

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Run the application with default settings
run: build
	@echo "Running model extractor..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Process models with custom input/output paths
process: build
	@echo "Processing models..."
	./$(BUILD_DIR)/$(BINARY_NAME) \
		--input $(MODELS_INDEX_PATH) \
		--output-dir output \
		--catalog-output $(CATALOG_OUTPUT_PATH)

# Quick development iteration
dev: fmt vet test build

# Full CI pipeline
ci: deps check test build

# Create release build
release: clean
	@echo "Creating release build..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Initialize go module (only run once)
init-module:
	@echo "Initializing Go module..."
	$(GOMOD) init github.com/chambridge/model-metadata-collection

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Generate documentation
docs:
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Documentation server starting at http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc not installed. Install it with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# Run security scan
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install it with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Setup development environment
setup:
	@echo "Setting up development environment..."
	$(GOMOD) download
	@echo "Installing development tools..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	$(GOGET) golang.org/x/tools/cmd/godoc@latest

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  clean       - Clean build artifacts and output"
	@echo "  test        - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  benchmark   - Run benchmarks"
	@echo "  lint        - Run linters"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  fmt-check   - Check code formatting"
	@echo "  deps        - Download dependencies"
	@echo "  check       - Run all checks (fmt-check, vet, lint)"
	@echo "  install     - Install binary to GOPATH/bin"
	@echo "  run         - Run with default settings"
	@echo "  process     - Run with custom input/output paths"
	@echo "  dev         - Quick development iteration"
	@echo "  ci          - Full CI pipeline"
	@echo "  release     - Create optimized release build"
	@echo "  update-deps - Update dependencies"
	@echo "  docs        - Generate documentation"
	@echo "  security    - Run security scan"
	@echo "  setup       - Setup development environment"
	@echo "  help        - Show this help"