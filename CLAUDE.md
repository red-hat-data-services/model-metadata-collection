# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application that extracts model metadata (specifically model cards) from Red Hat AI model container images. The tool has been enhanced to automatically consume data from HuggingFace collections and processes multiple ModelCar container images in parallel, scanning their layers for modelcard annotations, and extracts structured metadata with quality validation.

## Key Commands

### Build and Run
- `go run main.go` - Run the application to process HuggingFace collections and extract metadata from ModelCar container images
- `./scripts/update-collections.sh` - Update HuggingFace collection index files
- `go build` - Build the executable
- `go mod tidy` - Clean up dependencies
- `go fmt ./...` - Format Go code
- `go vet ./...` - Check for Go code issues
- `go test ./...` - Run tests (though no tests currently exist)

### Development
- `go mod download` - Download dependencies
- `gofmt -w .` - Format all Go files in place

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

## Current Capabilities
- ✅ HuggingFace collections integration with automatic discovery
- ✅ Semver version detection and handling
- ✅ High-quality metadata extraction with validation
- ✅ Date-to-epoch conversion
- ✅ OCI artifact extraction
- ✅ Parallel processing with concurrency limiting
- ✅ Graceful error handling and fallbacks
- ✅ Version-specific index file generation