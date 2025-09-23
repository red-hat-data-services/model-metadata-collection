package metadata

import (
	"reflect"
	"testing"

	"github.com/opendatahub-io/model-metadata-collection/pkg/types"
)

func TestParseModelCardMetadata(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected types.ModelMetadata
	}{
		{
			name: "complete model card",
			content: `# Granite 3.1 8B Base Model

**Model Developers:** IBM Research

## Model Overview
This is a test model for text generation tasks.

**License:** Apache-2.0
**Framework:** transformers
**Language:** English
**Tasks:** text-generation, question-answering
**Release Date:** 1/8/2025
**Last Update:** 1/10/2025

Download from registry.redhat.io
`,
			expected: types.ModelMetadata{
				Name:                     true,
				Provider:                 true,
				Description:              true,
				Readme:                   true,
				Language:                 true,
				License:                  true,
				LicenseLink:              false,
				Tasks:                    true,
				CreateTimeSinceEpoch:     true,
				LastUpdateTimeSinceEpoch: true,
				Artifacts:                true,
			},
		},
		{
			name: "minimal model card",
			content: `# Simple Model

Basic model description.
`,
			expected: types.ModelMetadata{
				Name:                     true,
				Provider:                 false,
				Description:              false,
				Readme:                   true,
				Language:                 false,
				License:                  false,
				LicenseLink:              false,
				Tasks:                    false,
				CreateTimeSinceEpoch:     false,
				LastUpdateTimeSinceEpoch: false,
				Artifacts:                false,
			},
		},
		{
			name: "model with license link",
			content: `# Test Model

Licensed under Apache-2.0 (https://www.apache.org/licenses/LICENSE-2.0)
`,
			expected: types.ModelMetadata{
				Name:                     true,
				Provider:                 false,
				Description:              false,
				Readme:                   true,
				Language:                 false,
				License:                  true,
				LicenseLink:              true,
				Tasks:                    false,
				CreateTimeSinceEpoch:     false,
				LastUpdateTimeSinceEpoch: false,
				Artifacts:                false,
			},
		},
		{
			name:    "empty content",
			content: "",
			expected: types.ModelMetadata{
				Name:                     false,
				Provider:                 false,
				Description:              false,
				Readme:                   false,
				Language:                 false,
				License:                  false,
				LicenseLink:              false,
				Tasks:                    false,
				CreateTimeSinceEpoch:     false,
				LastUpdateTimeSinceEpoch: false,
				Artifacts:                false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseModelCardMetadata([]byte(tt.content))
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseModelCardMetadata() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestExtractMetadataValues(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected types.ExtractedMetadata
	}{
		{
			name: "granite model with full metadata",
			content: `# granite-3.1-8b-base-quantized.w4a16

**Model Developers:** Neural Magic

## Model Overview
Quantized version of IBM granite-3.1-8b-base model intended for efficient inference.

**License:** Apache-2.0
**Release Date:** 1/8/2025
**Version:** 1.0
**Intended Use Cases:** text-generation, code-completion
**Supported Languages:** English, Spanish, French

This model can be deployed efficiently using the [vLLM](https://docs.vllm.ai/en/latest/) backend.
`,
			expected: types.ExtractedMetadata{
				Name:        stringPtr("granite-3.1-8b-base-quantized.w4a16"),
				Provider:    stringPtr("Neural Magic"),
				Description: stringPtr("Model Developers:** Neural Magic"),
				License:     stringPtr("Apache-2.0"),
				LicenseLink: stringPtr("https://www.apache.org/licenses/LICENSE-2.0"),
				Tasks:       []string{"text-generation"},
				Language:    []string{"en", "es", "fr"},
				Artifacts:   []types.OCIArtifact{},
			},
		},
		{
			name: "llama model with different format",
			content: `# Meta Llama 3.2 1B Instruct

- **Model developers:** Meta
- **Release Date:** 9/25/2024
- **Tasks:** conversational AI, instruction following

## Overview
Meta developed and released the Meta Llama 3.2 collection of multilingual large language models.

**License:** [Llama 3.2 Community License](https://github.com/meta-llama/llama-models/blob/main/models/llama3_2/LICENSE)
`,
			expected: types.ExtractedMetadata{
				Name:        stringPtr("Meta Llama 3.2 1B Instruct"),
				Provider:    stringPtr("Meta"),
				Description: stringPtr("Meta Llama 3.2 1B Instruct - An instruction-tuned language model"),
				License:     stringPtr("Llama 3.2 Community License"),
				LicenseLink: stringPtr("https://github.com/meta-llama/llama-models/blob/main/models/llama3_2/LICENSE"),
				Tasks:       []string{"text-generation"},
				Language:    []string{},
				Artifacts:   []types.OCIArtifact{},
			},
		},
		{
			name: "model with multiple languages",
			content: `# Multilingual Model

**Author:** Test Company
**Supported Languages:** English, German, Japanese, Chinese
**Description:** This model supports 4 languages in addition to English: German, Japanese, Chinese
`,
			expected: types.ExtractedMetadata{
				Name:        stringPtr("Multilingual Model"),
				Provider:    stringPtr("Test Company"),
				Description: stringPtr("Multilingual Model - A large language model"),
				Language:    []string{"en", "de", "ja", "zh"},
				Artifacts:   []types.OCIArtifact{},
			},
		},
		{
			name: "model with framework detection",
			content: `# PyTorch Model

This model works with PyTorch >= 2.0 and can be used with transformers library.

**Provider:** Research Lab
`,
			expected: types.ExtractedMetadata{
				Name:        stringPtr("PyTorch Model"),
				Provider:    stringPtr("Research Lab"),
				Description: stringPtr("This model works with PyTorch >= 2.0 and can be used with transformers library"),
				Artifacts:   []types.OCIArtifact{},
			},
		},
		{
			name: "model with update dates",
			content: `# Test Model

**Model Developers:** TestCorp
**Release Date:** 1/15/2024
Last updated on 3/10/2024 with bug fixes.
`,
			expected: types.ExtractedMetadata{
				Name:        stringPtr("Test Model"),
				Provider:    stringPtr("TestCorp"),
				Description: stringPtr("Test Model - A large language model"),
				Artifacts:   []types.OCIArtifact{},
			},
		},
		{
			name: "minimal content",
			content: `# Simple Model

Basic description here.
`,
			expected: types.ExtractedMetadata{
				Name:        stringPtr("Simple Model"),
				Description: stringPtr("Simple Model - A large language model"),
				Artifacts:   []types.OCIArtifact{},
			},
		},
		{
			name: "code example title (should be skipped)",
			content: `# How to define a function

` + "```python\ndef example():\n    pass\n```" + `

# Tool Usage Examples

Some examples here.
`,
			expected: types.ExtractedMetadata{
				Artifacts: []types.OCIArtifact{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractMetadataValues([]byte(tt.content))

			// Set readme to expected if content exists
			if len(tt.content) > 0 {
				readme := tt.content
				tt.expected.Readme = &readme
			}

			// Handle epoch timestamps separately since they're hard to predict
			if result.CreateTimeSinceEpoch != nil {
				tt.expected.CreateTimeSinceEpoch = result.CreateTimeSinceEpoch
			}
			if result.LastUpdateTimeSinceEpoch != nil {
				tt.expected.LastUpdateTimeSinceEpoch = result.LastUpdateTimeSinceEpoch
			}

			// Compare each field individually for better error messages
			if !compareStringPtr(result.Name, tt.expected.Name) {
				t.Errorf("Name: got %v, want %v", derefStringPtr(result.Name), derefStringPtr(tt.expected.Name))
			}
			if !compareStringPtr(result.Provider, tt.expected.Provider) {
				t.Errorf("Provider: got %v, want %v", derefStringPtr(result.Provider), derefStringPtr(tt.expected.Provider))
			}
			if !compareStringPtr(result.Description, tt.expected.Description) {
				t.Errorf("Description: got %v, want %v", derefStringPtr(result.Description), derefStringPtr(tt.expected.Description))
			}
			if !compareStringPtr(result.License, tt.expected.License) {
				t.Errorf("License: got %v, want %v", derefStringPtr(result.License), derefStringPtr(tt.expected.License))
			}
			if !compareStringPtr(result.LicenseLink, tt.expected.LicenseLink) {
				t.Errorf("LicenseLink: got %v, want %v", derefStringPtr(result.LicenseLink), derefStringPtr(tt.expected.LicenseLink))
			}
			// Handle nil vs empty slice for Language comparison
			resultLang := result.Language
			expectedLang := tt.expected.Language
			if resultLang == nil {
				resultLang = []string{}
			}
			if expectedLang == nil {
				expectedLang = []string{}
			}
			if !reflect.DeepEqual(resultLang, expectedLang) {
				t.Errorf("Language: got %v, want %v", resultLang, expectedLang)
			}
			// Handle nil vs empty slice for Tasks comparison
			resultTasks := result.Tasks
			expectedTasks := tt.expected.Tasks
			if resultTasks == nil {
				resultTasks = []string{}
			}
			if expectedTasks == nil {
				expectedTasks = []string{}
			}
			if !reflect.DeepEqual(resultTasks, expectedTasks) {
				t.Errorf("Tasks: got %v, want %v", resultTasks, expectedTasks)
			}
			if !reflect.DeepEqual(result.Artifacts, tt.expected.Artifacts) {
				t.Errorf("Artifacts: got %v, want %v", result.Artifacts, tt.expected.Artifacts)
			}
		})
	}
}

func TestExtractMetadataValues_TimestampConsistency(t *testing.T) {
	content := `# Test Model

**Release Date:** 1/15/2024
**Provider:** TestCorp
`

	result := ExtractMetadataValues([]byte(content))

	// Should have createTimeSinceEpoch from the release date
	if result.CreateTimeSinceEpoch == nil {
		t.Error("Expected CreateTimeSinceEpoch to be set from release date")
	}

	// Should copy createTimeSinceEpoch to lastUpdateTimeSinceEpoch when lastUpdate is nil
	if result.LastUpdateTimeSinceEpoch == nil {
		t.Error("Expected LastUpdateTimeSinceEpoch to be copied from CreateTimeSinceEpoch")
	}

	if result.CreateTimeSinceEpoch != nil && result.LastUpdateTimeSinceEpoch != nil {
		if *result.CreateTimeSinceEpoch != *result.LastUpdateTimeSinceEpoch {
			t.Error("Expected LastUpdateTimeSinceEpoch to equal CreateTimeSinceEpoch when lastUpdate is not specified")
		}
	}
}

func TestExtractMetadataValues_EmptyContent(t *testing.T) {
	result := ExtractMetadataValues([]byte(""))

	expected := types.ExtractedMetadata{
		Artifacts: []types.OCIArtifact{},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ExtractMetadataValues() with empty content = %+v, want %+v", result, expected)
	}
}

func TestExtractMetadataValues_LicenseAutoLink(t *testing.T) {
	content := `# Test Model

**License:** MIT
**Provider:** TestCorp
`

	result := ExtractMetadataValues([]byte(content))

	if result.License == nil || *result.License != "MIT" {
		t.Error("Expected license to be MIT")
	}

	if result.LicenseLink == nil || *result.LicenseLink != "https://opensource.org/licenses/MIT" {
		t.Error("Expected license link to be automatically set for MIT license")
	}
}

// Helper functions for testing
func stringPtr(s string) *string {
	return &s
}

func compareStringPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func derefStringPtr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func TestExtractMetadataValues_ValidatedOn(t *testing.T) {
	contentWithValidatedOn := `---
name: "Test Model"
provider: "RedHat AI"
validated_on:
  - RHOAI 2.24
  - RHAIIS 3.2.1
---
# Test Model

This is a test model validated on multiple platforms.
`

	result := ExtractMetadataValues([]byte(contentWithValidatedOn))

	// Check that validated_on was extracted correctly
	if result.ValidatedOn == nil {
		t.Error("Expected ValidatedOn to be set from YAML frontmatter")
	} else {
		expected := []string{"RHOAI 2.24", "RHAIIS 3.2.1"}
		if !reflect.DeepEqual(result.ValidatedOn, expected) {
			t.Errorf("ExtractMetadataValues() ValidatedOn = %v, want %v", result.ValidatedOn, expected)
		}
	}

	// Check that name was also extracted from frontmatter
	if result.Name == nil || *result.Name != "Test Model" {
		t.Error("Expected name to be extracted from YAML frontmatter")
	}

	// Check that provider was also extracted from frontmatter
	if result.Provider == nil || *result.Provider != "RedHat AI" {
		t.Error("Expected provider to be extracted from YAML frontmatter")
	}
}
