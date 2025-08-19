package utils

import (
	"reflect"
	"testing"
)

func TestParseLanguageNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single language",
			input:    "english",
			expected: []string{"en"},
		},
		{
			name:     "multiple languages with comma",
			input:    "english, spanish, french",
			expected: []string{"en", "es", "fr"},
		},
		{
			name:     "multiple languages with and",
			input:    "english and spanish",
			expected: []string{"en", "es"},
		},
		{
			name:     "unknown language",
			input:    "klingon",
			expected: nil,
		},
		{
			name:     "mixed case",
			input:    "ENGLISH, Spanish",
			expected: []string{"en", "es"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLanguageNames(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseLanguageNames() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateDescriptionFromModelName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic model name",
			input:    "RedHatAI/Llama-3.3-70B-Instruct",
			expected: "Llama 3.3 70B Instruct",
		},
		{
			name:     "quantized model",
			input:    "RedHatAI/granite-3.1-8b-base-quantized.w4a16",
			expected: "Granite 3.1 8b Base (w4a16 quantized)",
		},
		{
			name:     "whisper model",
			input:    "whisper-large-v2-quantized.w4a16",
			expected: "Whisper large v2 (w4a16 quantized)",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "fp8 dynamic model",
			input:    "meta-llama/Llama-3.1-8B-Instruct-FP8-dynamic",
			expected: "Llama 3.1 8B Instruct ((FP8) dynamic)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateDescriptionFromModelName(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateDescriptionFromModelName() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestNormalizeModelName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "registry reference",
			input:    "registry.redhat.io/rhelai1/modelcar-granite-3-1-8b:1.0",
			expected: "granite-3-1-8b",
		},
		{
			name:     "huggingface reference",
			input:    "RedHatAI/Llama-3.3-70B-Instruct",
			expected: "llama-3-3-70b-instruct",
		},
		{
			name:     "mixed separators",
			input:    "test_model.name with spaces",
			expected: "test-model-name-with-spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeModelName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeModelName() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
	}{
		{
			name:     "exact match",
			s1:       "test-model",
			s2:       "test-model",
			expected: 1.0,
		},
		{
			name:     "one contains other",
			s1:       "test-model-v1",
			s2:       "test-model",
			expected: 0.8,
		},
		{
			name:     "no similarity",
			s1:       "completely-different",
			s2:       "model-name",
			expected: 0.0,
		},
		{
			name:     "partial similarity",
			s1:       "granite-3-1-8b",
			s2:       "granite-8b-model",
			expected: 0.5, // 2 common tokens out of 4 max tokens
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSimilarity(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("CalculateSimilarity() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
