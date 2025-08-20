# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application that extracts model metadata (specifically model cards) from Red Hat AI model container images. The tool has been enhanced to automatically consume data from HuggingFace collections and processes multiple ModelCar container images in parallel, scanning their layers for modelcard annotations, and extracts structured metadata with quality validation.

## Key Commands

### Build and Run
- `make build` - Build the model-extractor binary
- `make run` - Run the application to process HuggingFace collections and extract metadata from ModelCar container images
- `make clean` - Clean build artifacts and output
- `make fmt` - Format Go code
- `make vet` - Check for Go code issues
- `make test` - Run tests with coverage reporting
- `make docker-build` - Build Docker container image

### Development
- `make deps` - Download and tidy dependencies
- `make dev` - Quick development iteration (fmt, vet, test, build)
- `make ci` - Full CI pipeline
- `make lint` - Run linters (requires golangci-lint)

## Architecture

### Core Components

1. **HuggingFace Collections Processing** (`processHuggingFaceCollections`)
   - Automatically discovers Red Hat AI validated model collections
   - Supports semver version detection (v1.0, v2.1, etc.)
   - Generates version-specific index files in `data/hugging-face-redhat-ai-validated-v{version}.yaml`
   - Falls back to known collections if discovery fails

2. **Main Processing Flow** (`main.go:654+`)
   - First processes HuggingFace collections to generate/update index files
   - Uses latest version index file or falls back to legacy `models-index.yaml`
   - Processes ModelCar manifest references in parallel with semaphore limiting (max 5 concurrent processes)
   - Each goroutine handles one manifest reference from start to finish

3. **Container Image Processing** (`FetchManifestSrcAndLayers`)
   - Uses `github.com/containers/image/v5` library for OCI container operations
   - Parses Docker references and creates image sources
   - Extracts manifest metadata and layer information

4. **ModelCard Extraction and Metadata Processing** (`ScanLayersForModelCarD`)
   - Scans container layers for `io.opendatahub.modelcar.layer.type: "modelcard"` annotation
   - Handles both gzipped and uncompressed tar layers
   - Extracts `.md` files and processes metadata with quality validation
   - Converts dates to Unix epoch timestamps
   - Extracts OCI artifacts and model properties
   - Writes structured metadata alongside modelcard content

5. **Output Organization**
   - Creates sanitized directory names from manifest references
   - Generates structured YAML metadata files with extracted information
   - Output structure: `output/{sanitized-manifest-ref}/models/{modelcard.md,metadata.yaml}`
   - Creates summary `manifests.yaml` with metadata presence indicators

### Key Dependencies
- `github.com/containers/image/v5` - Container image manipulation and OCI registry access
- Standard library: `archive/tar`, `compress/gzip`, `sync` for concurrent processing

### Concurrency Model
- Uses `sync.WaitGroup` to coordinate parallel processing
- Semaphore pattern with buffered channel limits concurrent goroutines to 5
- Each manifest reference is processed independently in its own goroutine

### Data Flow
1. HuggingFace collections discovery → Version-specific index file generation
2. Load model references from latest index or legacy file → Parallel processing
3. Fetch container manifest and layers → Layer scanning
4. Find modelcard-annotated layers → Extract and validate metadata
5. Generate structured YAML metadata → Write organized output

## Enhanced Features (Recent Updates)

### HuggingFace Collections Integration
- **Automatic Discovery**: Scans HuggingFace for Red Hat AI validated model collections
- **Version Support**: Handles semver patterns like "Red Hat AI validated models - v1.0"
- **Index Generation**: Creates `data/hugging-face-redhat-ai-validated-v{version}.yaml` files with model names, URLs, and README paths
- **Flexible Processing**: Falls back gracefully when HuggingFace APIs are unavailable

### Metadata Quality Improvements
- **Date Conversion**: Converts date strings to Unix epoch timestamps (e.g., "7/11/2024" → 1720656000)
- **OCI Artifact Extraction**: Focuses on relevant container registry patterns and model names
- **Validation Functions**: `isValidValue()` and `cleanExtractedValue()` ensure quality metadata
- **Structured Output**: Generates both `modelcard.md` and `metadata.yaml` for each model

### Version-Specific Index Files
```yaml
# Example: data/hugging-face-redhat-ai-validated-v1-0.yaml
version: v1.0
models:
  - name: RedHatAI/Llama-4-Scout-17B-16E-Instruct
    url: https://huggingface.co/RedHatAI/Llama-4-Scout-17B-16E-Instruct
    readme_path: /RedHatAI/Llama-4-Scout-17B-16E-Instruct/README.md
```

## Docker Build and Deployment

### Prerequisites
Before building the Docker image, you must authenticate with Red Hat's container registry:

```bash
# Login to Red Hat container registry (required for accessing ModelCar images)
docker login registry.redhat.io
# Enter your Red Hat account credentials when prompted
```

### Building the Container
The Docker build uses a multi-stage approach:

1. **Builder Stage**: Uses `registry.access.redhat.com/ubi9/go-toolset:1.24` to compile the Go application
2. **Generator Stage**: Uses `registry.access.redhat.com/ubi9-minimal` to run model extraction and generate `data/models-catalog.yaml`
3. **Runtime Stage**: Uses `registry.access.redhat.com/ubi9-micro` for the final minimal image containing only the catalog

```bash
# Build the Docker image (requires prior registry.redhat.io authentication)
make docker-build

# The build automatically uses your host Docker authentication
# No additional credential files needed
```

### Authentication Details
- The build process automatically detects and uses your existing `~/.docker/config.json` authentication
- Registry credentials are securely mounted during build using Docker BuildKit secrets
- No credentials are stored in the final container image
- If no authentication is found, the build gracefully continues without registry access

### Usage Examples
```bash
# Run container and access the generated catalog
docker run -d --name model-metadata-catalog model-metadata-collection:latest

# Copy catalog from container to host
docker cp model-metadata-catalog:/app/data/models-catalog.yaml ./models-catalog.yaml

# Mount catalog directory for external access
docker run -d -v $(pwd)/catalog-data:/app/data --name catalog model-metadata-collection:latest
```

## Current Capabilities
- ✅ HuggingFace collections integration with automatic discovery
- ✅ Semver version detection and handling
- ✅ High-quality metadata extraction with validation
- ✅ Date-to-epoch conversion
- ✅ OCI artifact extraction
- ✅ Parallel processing with concurrency limiting
- ✅ Graceful error handling and fallbacks
- ✅ Version-specific index file generation
- ✅ Multi-stage Docker builds with registry authentication
- ✅ Containerized catalog generation
