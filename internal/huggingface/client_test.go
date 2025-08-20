package huggingface

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/chambridge/model-metadata-collection/pkg/types"
)

func TestFetchCollections(t *testing.T) {
	// Test basic function structure - network calls will likely fail in test environment
	// but we can test that the function returns an appropriate error
	_, err := FetchCollections()
	if err != nil {
		// Expected to fail due to network unavailability in tests
		// Accept various types of network/API errors
		if !strings.Contains(err.Error(), "failed to fetch collections") &&
			!strings.Contains(err.Error(), "failed to parse collections") &&
			!strings.Contains(err.Error(), "json:") {
			t.Errorf("Expected a collections-related error, got: %v", err)
		}
	}
}

func TestFetchCollectionDetails(t *testing.T) {
	// Test with a test collection ID
	_, err := FetchCollectionDetails("test-collection-id")
	if err != nil {
		// Expected to fail due to network unavailability in tests
		if !strings.Contains(err.Error(), "failed to fetch collection details") {
			t.Errorf("Expected 'failed to fetch collection details' error, got: %v", err)
		}
	}
}

func TestDiscoverValidatedModelCollections(t *testing.T) {
	// Test basic function structure
	_, err := DiscoverValidatedModelCollections()
	if err != nil {
		// Expected to fail due to network unavailability in tests
		// Accept various types of network/API errors
		if !strings.Contains(err.Error(), "failed to fetch user collections") &&
			!strings.Contains(err.Error(), "failed to parse collections") &&
			!strings.Contains(err.Error(), "json:") {
			t.Errorf("Expected a collections-related error, got: %v", err)
		}
	}
}

func TestFetchModelDetails(t *testing.T) {
	// Test with a test model name
	_, err := FetchModelDetails("test/model")
	if err != nil {
		// Expected to fail due to network unavailability in tests
		// Accept various types of network/API errors
		if !strings.Contains(err.Error(), "failed to fetch model details") &&
			!strings.Contains(err.Error(), "API returned status") {
			t.Errorf("Expected a model details related error, got: %v", err)
		}
	}
}

func TestFetchReadme(t *testing.T) {
	// Test with a test model name
	_, err := FetchReadme("test/model")
	if err != nil {
		// Expected to fail due to network unavailability in tests
		// Accept various types of network/API errors
		if !strings.Contains(err.Error(), "failed to fetch README") &&
			!strings.Contains(err.Error(), "README not found") &&
			!strings.Contains(err.Error(), "status") {
			t.Errorf("Expected a README related error, got: %v", err)
		}
	}
}

func TestGetLatestVersionIndexFile(t *testing.T) {
	// Test with no files
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		err := os.Chdir(originalDir)
		if err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	err = os.MkdirAll("data", 0755)
	if err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	// Test with no version files
	_, err = GetLatestVersionIndexFile()
	if err == nil {
		t.Error("Expected error when no version index files exist")
	}

	// Create some test version files
	testFiles := []string{
		"data/hugging-face-redhat-ai-validated-v1-0.yaml",
		"data/hugging-face-redhat-ai-validated-v2-0.yaml",
		"data/hugging-face-redhat-ai-validated-v1-5.yaml",
	}

	for _, file := range testFiles {
		err = os.WriteFile(file, []byte("version: test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Test getting latest version file
	latest, err := GetLatestVersionIndexFile()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return the last file alphabetically (v2-0)
	expected := "data/hugging-face-redhat-ai-validated-v2-0.yaml"
	if latest != expected {
		t.Errorf("Expected %s, got %s", expected, latest)
	}
}

func TestLoadModelsFromVersionIndex(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-version-index.yaml")

	tests := []struct {
		name        string
		fileContent types.VersionIndex
		expected    []string
		expectError bool
	}{
		{
			name: "valid version index",
			fileContent: types.VersionIndex{
				Version: "v1.0",
				Models: []types.ModelIndex{
					{
						Name:       "RedHatAI/granite-3.1-8b-base",
						URL:        "https://huggingface.co/RedHatAI/granite-3.1-8b-base",
						ReadmePath: "/RedHatAI/granite-3.1-8b-base/README.md",
					},
					{
						Name:       "microsoft/Phi-3.5-mini-instruct",
						URL:        "https://huggingface.co/microsoft/Phi-3.5-mini-instruct",
						ReadmePath: "/microsoft/Phi-3.5-mini-instruct/README.md",
					},
				},
			},
			expected: []string{
				"registry.redhat.io/rhelai1/modelcar-redhatai-granite-3.1-8b-base",
				"registry.redhat.io/rhelai1/modelcar-microsoft-phi-3.5-mini-instruct",
			},
			expectError: false,
		},
		{
			name: "empty models",
			fileContent: types.VersionIndex{
				Version: "v1.0",
				Models:  []types.ModelIndex{},
			},
			expected:    []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test data to file
			data, err := yaml.Marshal(tt.fileContent)
			if err != nil {
				t.Fatalf("Failed to marshal test data: %v", err)
			}

			err = os.WriteFile(tmpFile, data, 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Test the function
			result, err := LoadModelsFromVersionIndex(tmpFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d models, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected model[%d] = %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestLoadModelsFromVersionIndex_FileNotFound(t *testing.T) {
	_, err := LoadModelsFromVersionIndex("nonexistent-file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestLoadModelsFromVersionIndex_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")

	// Create invalid YAML
	err := os.WriteFile(tmpFile, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	_, err = LoadModelsFromVersionIndex(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestExtractProviderFromReadme(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "model developers field",
			content: `# Test Model

**Model Developers:** Neural Magic

Some other content here.
`,
			expected: "Neural Magic",
		},
		{
			name: "author field",
			content: `# Test Model

**Author:** IBM Research

Some content.
`,
			expected: "IBM Research",
		},
		{
			name: "provider field",
			content: `# Test Model

**Provider:** Microsoft

Content here.
`,
			expected: "Microsoft",
		},
		{
			name: "no provider found",
			content: `# Test Model

Some content without any company information.
`,
			expected: "",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
		{
			name: "provider with special characters",
			content: `# Test Model

**Model Developers:** Red Hat, Inc.

Content.
`,
			expected: "Red Hat, Inc",
		},
		{
			name: "invalid provider (too short)",
			content: `# Test Model

**Author:** X

Content.
`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractProviderFromReadme(tt.content)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
