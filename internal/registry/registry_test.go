package registry

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseRegistryImageRef(t *testing.T) {
	tests := []struct {
		name               string
		imageRef           string
		expectedRegistry   string
		expectedRepository string
		expectedImageName  string
		expectedTag        string
		expectError        bool
	}{
		{
			name:               "standard registry reference with tag",
			imageRef:           "registry.redhat.io/rhelai1/modelcar-granite-3-1-8b-base:1.0",
			expectedRegistry:   "registry.redhat.io",
			expectedRepository: "rhelai1",
			expectedImageName:  "modelcar-granite-3-1-8b-base",
			expectedTag:        "1.0",
			expectError:        false,
		},
		{
			name:               "reference without tag (defaults to latest)",
			imageRef:           "registry.redhat.io/rhelai1/modelcar-granite-3-1-8b-base",
			expectedRegistry:   "registry.redhat.io",
			expectedRepository: "rhelai1",
			expectedImageName:  "modelcar-granite-3-1-8b-base",
			expectedTag:        "latest",
			expectError:        false,
		},
		{
			name:               "complex image name with multiple segments",
			imageRef:           "registry.redhat.io/rhelai1/deep/nested/modelcar-complex:2.1",
			expectedRegistry:   "registry.redhat.io",
			expectedRepository: "rhelai1",
			expectedImageName:  "deep/nested/modelcar-complex",
			expectedTag:        "2.1",
			expectError:        false,
		},
		{
			name:               "localhost registry",
			imageRef:           "localhost:5000/test/simple-model:latest",
			expectedRegistry:   "localhost:5000",
			expectedRepository: "test",
			expectedImageName:  "simple-model",
			expectedTag:        "latest",
			expectError:        false,
		},
		{
			name:        "invalid format - too few parts",
			imageRef:    "registry.io/image",
			expectError: true,
		},
		{
			name:        "invalid format - single part",
			imageRef:    "image",
			expectError: true,
		},
		{
			name:        "empty string",
			imageRef:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, repository, imageName, tag, err := parseRegistryImageRef(tt.imageRef)

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

			if registry != tt.expectedRegistry {
				t.Errorf("Registry: got %s, want %s", registry, tt.expectedRegistry)
			}
			if repository != tt.expectedRepository {
				t.Errorf("Repository: got %s, want %s", repository, tt.expectedRepository)
			}
			if imageName != tt.expectedImageName {
				t.Errorf("ImageName: got %s, want %s", imageName, tt.expectedImageName)
			}
			if tag != tt.expectedTag {
				t.Errorf("Tag: got %s, want %s", tag, tt.expectedTag)
			}
		})
	}
}

func TestFetchRegistryMetadata(t *testing.T) {
	t.Skip("Skipping integration test that makes network calls - should be run separately with -integration flag")
	tests := []struct {
		name             string
		imageRef         string
		expectError      bool
		expectedURIStart string
		checkTimestamp   bool
	}{
		{
			name:             "valid red hat registry reference",
			imageRef:         "registry.redhat.io/rhelai1/modelcar-granite-3-1-8b-base:1.0",
			expectError:      false,
			expectedURIStart: "oci://registry.redhat.io/rhelai1/modelcar-granite-3-1-8b-base:1.0",
			checkTimestamp:   true,
		},
		{
			name:             "non-red hat registry",
			imageRef:         "docker.io/library/alpine:latest",
			expectError:      false,
			expectedURIStart: "oci://docker.io/library/alpine:latest",
			checkTimestamp:   true,
		},
		{
			name:        "invalid image reference",
			imageRef:    "invalid/ref",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FetchRegistryMetadata(tt.imageRef)

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

			if result == nil {
				t.Fatal("Result should not be nil")
			}

			// Check URI format
			if result.URI != tt.expectedURIStart {
				t.Errorf("URI: got %s, want %s", result.URI, tt.expectedURIStart)
			}

			// Check timestamps if required
			if tt.checkTimestamp {
				if result.CreateTimeSinceEpoch == nil {
					t.Error("CreateTimeSinceEpoch should not be nil")
				}
				if result.LastUpdateTimeSinceEpoch == nil {
					t.Error("LastUpdateTimeSinceEpoch should not be nil")
				}

				// Timestamps should be recent (within last hour for test)
				currentTime := time.Now().Unix() * 1000
				hourAgo := currentTime - (60 * 60 * 1000)

				if result.CreateTimeSinceEpoch != nil && *result.CreateTimeSinceEpoch < hourAgo {
					// Only warn if the timestamp seems too old (might be from actual registry)
					t.Logf("Warning: CreateTimeSinceEpoch seems old: %d", *result.CreateTimeSinceEpoch)
				}
			}

			// Check custom properties
			if result.CustomProperties == nil {
				t.Error("CustomProperties should not be nil")
			}

			// Verify source property exists
			if source, exists := result.CustomProperties["source"]; exists {
				if sourceMap, ok := source.(map[string]interface{}); ok {
					if sourceValue, exists := sourceMap["string_value"]; exists {
						if sourceStr, ok := sourceValue.(string); ok && sourceStr != "" {
							t.Logf("Source: %s", sourceStr)
						} else {
							t.Error("Source string_value should be a non-empty string")
						}
					} else {
						t.Error("Source should have string_value field")
					}
				} else {
					t.Error("Source should be a map")
				}
			} else {
				t.Error("CustomProperties should contain source")
			}

			// Verify type property exists
			if typeVal, exists := result.CustomProperties["type"]; exists {
				if typeMap, ok := typeVal.(map[string]interface{}); ok {
					if typeValue, exists := typeMap["string_value"]; exists {
						if typeStr, ok := typeValue.(string); ok && typeStr == "modelcar" {
							t.Logf("Type: %s", typeStr)
						} else {
							t.Errorf("Type string_value should be 'modelcar', got: %v", typeValue)
						}
					} else {
						t.Error("Type should have string_value field")
					}
				} else {
					t.Error("Type should be a map")
				}
			} else {
				t.Error("CustomProperties should contain type")
			}
		})
	}
}

func TestExtractOCIArtifactsFromRegistry(t *testing.T) {
	t.Skip("Skipping integration test that makes network calls - should be run separately with -integration flag")
	tests := []struct {
		name            string
		manifestRef     string
		expectArtifacts int
		checkURI        string
	}{
		{
			name:            "valid registry reference",
			manifestRef:     "registry.redhat.io/rhelai1/modelcar-granite-3-1-8b-base:1.0",
			expectArtifacts: 1,
			checkURI:        "oci://registry.redhat.io/rhelai1/modelcar-granite-3-1-8b-base:1.0",
		},
		{
			name:            "non-red hat registry",
			manifestRef:     "docker.io/library/alpine:latest",
			expectArtifacts: 1,
			checkURI:        "oci://docker.io/library/alpine:latest",
		},
		{
			name:            "invalid reference - should still create artifact with error",
			manifestRef:     "invalid/ref",
			expectArtifacts: 0, // parseRegistryImageRef will fail, so no artifact created
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractOCIArtifactsFromRegistry(tt.manifestRef)

			if len(result) != tt.expectArtifacts {
				t.Errorf("Expected %d artifacts, got %d", tt.expectArtifacts, len(result))
				return
			}

			if tt.expectArtifacts > 0 {
				artifact := result[0]

				// Check URI
				if artifact.URI != tt.checkURI {
					t.Errorf("URI: got %s, want %s", artifact.URI, tt.checkURI)
				}

				// Check timestamps
				if artifact.CreateTimeSinceEpoch == nil {
					t.Error("CreateTimeSinceEpoch should not be nil")
				}
				if artifact.LastUpdateTimeSinceEpoch == nil {
					t.Error("LastUpdateTimeSinceEpoch should not be nil")
				}

				// Check custom properties
				if artifact.CustomProperties == nil {
					t.Error("CustomProperties should not be nil")
				}
			}
		})
	}
}

func TestFetchRegistryMetadata_ErrorHandling(t *testing.T) {
	t.Skip("Skipping integration test that makes network calls - should be run separately with -integration flag")
	// Test with a reference that will definitely fail network call
	// (using a non-existent domain to ensure network failure)
	imageRef := "nonexistent.registry.example.com/test/model:1.0"

	result, err := FetchRegistryMetadata(imageRef)
	if err != nil {
		t.Errorf("FetchRegistryMetadata should not return error for network failures, got: %v", err)
		return
	}

	if result == nil {
		t.Fatal("Result should not be nil even on network failure")
	}

	// Should create fallback artifact
	expectedURI := "oci://nonexistent.registry.example.com/test/model:1.0"
	if result.URI != expectedURI {
		t.Errorf("URI: got %s, want %s", result.URI, expectedURI)
	}

	// Should have timestamps
	if result.CreateTimeSinceEpoch == nil {
		t.Error("CreateTimeSinceEpoch should not be nil")
	}

	// Should have source in custom properties
	if source, exists := result.CustomProperties["source"]; exists {
		if sourceMap, ok := source.(map[string]interface{}); ok {
			if sourceValue, exists := sourceMap["string_value"]; exists {
				if sourceStr, ok := sourceValue.(string); ok {
					if sourceStr != "nonexistent.registry.example.com" {
						t.Errorf("Source should be registry name, got: %s", sourceStr)
					}
				}
			}
		}
	}
}

func TestRegistryManifest_StructureCompatibility(t *testing.T) {
	// Test that our RegistryManifest struct can handle typical registry JSON
	testJSON := `{
		"config": {
			"created": "2024-01-15T10:30:00Z"
		},
		"history": [
			{
				"created": "2024-01-15T10:30:00Z"
			},
			{
				"created": "2024-01-15T11:00:00Z"
			}
		],
		"annotations": {
			"io.opendatahub.modelcar.layer.type": "modelcard",
			"custom.annotation": "test-value"
		}
	}`

	var manifest RegistryManifest
	err := json.Unmarshal([]byte(testJSON), &manifest)
	if err != nil {
		t.Fatalf("Failed to unmarshal test JSON: %v", err)
	}

	// Verify structure
	if manifest.Config.Created != "2024-01-15T10:30:00Z" {
		t.Errorf("Config.Created: got %s, want %s", manifest.Config.Created, "2024-01-15T10:30:00Z")
	}

	if len(manifest.History) != 2 {
		t.Errorf("History length: got %d, want %d", len(manifest.History), 2)
	}

	if manifest.History[1].Created != "2024-01-15T11:00:00Z" {
		t.Errorf("History[1].Created: got %s, want %s", manifest.History[1].Created, "2024-01-15T11:00:00Z")
	}

	if len(manifest.Annotations) != 2 {
		t.Errorf("Annotations length: got %d, want %d", len(manifest.Annotations), 2)
	}

	if manifest.Annotations["io.opendatahub.modelcar.layer.type"] != "modelcard" {
		t.Errorf("Annotation value incorrect")
	}
}

func TestExtractOCIArtifactsFromRegistry_Properties(t *testing.T) {
	manifestRef := "registry.redhat.io/rhelai1/test-model:1.0"
	artifacts := ExtractOCIArtifactsFromRegistry(manifestRef)

	if len(artifacts) != 1 {
		t.Fatalf("Expected 1 artifact, got %d", len(artifacts))
	}

	artifact := artifacts[0]

	// Test custom properties structure
	requiredProps := []string{"source", "type"}
	for _, prop := range requiredProps {
		if val, exists := artifact.CustomProperties[prop]; exists {
			if propMap, ok := val.(map[string]interface{}); ok {
				if _, exists := propMap["string_value"]; !exists {
					t.Errorf("Property %s should have string_value field", prop)
				}
			} else {
				t.Errorf("Property %s should be a map[string]interface{}", prop)
			}
		} else {
			t.Errorf("Required property %s not found", prop)
		}
	}

	// Verify type is modelcar
	if typeVal, exists := artifact.CustomProperties["type"]; exists {
		if typeMap, ok := typeVal.(map[string]interface{}); ok {
			if stringVal, exists := typeMap["string_value"]; exists {
				if stringVal != "modelcar" {
					t.Errorf("Expected type 'modelcar', got %v", stringVal)
				}
			}
		}
	}
}

// Test to ensure artifacts slice is never nil
func TestExtractOCIArtifactsFromRegistry_NeverNil(t *testing.T) {
	// Even with invalid input, should return empty slice, not nil
	result := ExtractOCIArtifactsFromRegistry("completely/invalid")

	if result == nil {
		t.Error("Result should never be nil, should be empty slice instead")
	}

	// Should be empty due to parse error
	if len(result) != 0 {
		t.Errorf("Expected empty slice for invalid input, got %d artifacts", len(result))
	}
}
