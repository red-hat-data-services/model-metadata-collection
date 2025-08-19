package utils

import (
	"testing"
	"time"
)

func TestIsValidValue(t *testing.T) {
	tests := []struct {
		name            string
		value           string
		minLength       int
		maxLength       int
		allowedPatterns []string
		expected        bool
	}{
		{
			name:      "valid basic string",
			value:     "test",
			minLength: 1,
			maxLength: 10,
			expected:  true,
		},
		{
			name:      "string too short",
			value:     "a",
			minLength: 2,
			maxLength: 10,
			expected:  false,
		},
		{
			name:      "string too long",
			value:     "this is a very long string that exceeds the limit",
			minLength: 1,
			maxLength: 10,
			expected:  false,
		},
		{
			name:            "pattern match success",
			value:           "test123",
			minLength:       1,
			maxLength:       10,
			allowedPatterns: []string{`^[a-z]+\d+$`},
			expected:        true,
		},
		{
			name:            "pattern match failure",
			value:           "TEST123",
			minLength:       1,
			maxLength:       10,
			allowedPatterns: []string{`^[a-z]+\d+$`},
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidValue(tt.value, tt.minLength, tt.maxLength, tt.allowedPatterns)
			if result != tt.expected {
				t.Errorf("IsValidValue() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCleanExtractedValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic cleanup",
			input:    "  test value  ",
			expected: "test value",
		},
		{
			name:     "markdown formatting removal",
			input:    "**bold text**",
			expected: "bold text",
		},
		{
			name:     "trailing punctuation removal",
			input:    "test value:",
			expected: "test value",
		},
		{
			name:     "combined cleanup",
			input:    "  *_`\"test value`\"_*:  ",
			expected: "test value`\"_*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanExtractedValue(tt.input)
			if result != tt.expected {
				t.Errorf("CleanExtractedValue() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeManifestRef(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic registry reference",
			input:    "registry.redhat.io/rhelai1/modelcar-granite:1.0",
			expected: "registry.redhat.io_rhelai1_modelcar-granite_1.0",
		},
		{
			name:     "complex reference with multiple special chars",
			input:    "registry.io/path/with:colon/and\\backslash?question",
			expected: "registry.io_path_with_colon_and_backslash_question",
		},
		{
			name:     "multiple underscores cleanup",
			input:    "test///multiple\\\\\\slashes",
			expected: "test_multiple_slashes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeManifestRef(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeManifestRef() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestParseDateToEpoch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *int64
	}{
		{
			name:     "MM/DD/YYYY format",
			input:    "01/15/2024",
			expected: int64Ptr(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).Unix()),
		},
		{
			name:     "YYYY-MM-DD format",
			input:    "2024-01-15",
			expected: int64Ptr(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).Unix()),
		},
		{
			name:     "invalid date format",
			input:    "invalid-date",
			expected: nil,
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDateToEpoch(tt.input)
			if (result == nil) != (tt.expected == nil) {
				t.Errorf("ParseDateToEpoch() = %v, expected %v", result, tt.expected)
				return
			}
			if result != nil && tt.expected != nil && *result != *tt.expected {
				t.Errorf("ParseDateToEpoch() = %v, expected %v", *result, *tt.expected)
			}
		})
	}
}

func TestParseTimeToEpochInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *int64
	}{
		{
			name:     "RFC3339 format",
			input:    "2024-01-15T10:30:00Z",
			expected: int64Ptr(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).Unix() * 1000),
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "invalid format",
			input:    "invalid-time",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTimeToEpochInt64(tt.input)
			if (result == nil) != (tt.expected == nil) {
				t.Errorf("ParseTimeToEpochInt64() = %v, expected %v", result, tt.expected)
				return
			}
			if result != nil && tt.expected != nil && *result != *tt.expected {
				t.Errorf("ParseTimeToEpochInt64() = %v, expected %v", *result, *tt.expected)
			}
		})
	}
}

// Helper function to create int64 pointer
func int64Ptr(i int64) *int64 {
	return &i
}
