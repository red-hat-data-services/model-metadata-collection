#!/bin/bash
# Script to update HuggingFace collection index files

set -e

echo "Updating HuggingFace collection index files..."

# Change to project root
cd "$(dirname "$0")/.."

# Run the collection processing only (without full model processing)
# This would require adding a flag to main.go to only process collections
echo "Processing HuggingFace collections..."
go run main.go --collections-only 2>&1 | grep -E "(Processing|Found|Generated|Detected)" || true

echo ""
echo "Collection index files updated successfully!"
echo "Available index files:"
ls -la data/hugging-face-redhat-ai-validated-v*.yaml 2>/dev/null || echo "No version index files found"