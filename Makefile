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

# Docker parameters
DOCKER_IMAGE_NAME?=model-metadata-collection
DOCKER_IMAGE_TAG?=latest
DOCKER_FULL_IMAGE_NAME=$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)

.PHONY: all build clean test test-coverage lint fmt vet deps check help run process docker-build

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

# Docker targets

# Build Docker image
docker-build:
	@echo "Building Docker image: $(DOCKER_FULL_IMAGE_NAME)"
	@echo "Using host Docker authentication for registry access..."
	@# Use buildkit to mount Docker config for registry authentication
	@if [ -f ~/.docker/config.json ]; then \
		echo "Found Docker config, using host authentication"; \
		DOCKER_BUILDKIT=1 docker build \
			--secret id=dockerconfig,src=$$HOME/.docker/config.json \
			--build-arg BUILDKIT_INLINE_CACHE=1 \
			-t $(DOCKER_FULL_IMAGE_NAME) .; \
	else \
		echo "No Docker config found, building without registry authentication"; \
		docker build -t $(DOCKER_FULL_IMAGE_NAME) .; \
	fi
	@echo "Build completed successfully!"
	@echo "Image: $(DOCKER_FULL_IMAGE_NAME)"
	@echo ""
	@echo "Image details:"
	@docker images $(DOCKER_IMAGE_NAME) --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedSince}}\t{{.Size}}"
	@echo ""
	@echo "Usage examples:"
	@echo "  # Run container:"
	@echo "  docker run -d --name model-metadata-catalog $(DOCKER_FULL_IMAGE_NAME)"
	@echo ""
	@echo "  # Copy catalog from container:"
	@echo "  docker cp model-metadata-catalog:/app/data/models-catalog.yaml ./models-catalog.yaml"
	@echo ""
	@echo "  # Mount catalog directory:"
	@echo "  docker run -d -v \$$(pwd)/catalog-data:/app/data --name catalog $(DOCKER_FULL_IMAGE_NAME)"
	
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
	@echo "  docker-build - Build Docker image"
	@echo "  help         - Show this help"
