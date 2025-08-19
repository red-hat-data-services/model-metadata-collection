package utils

import (
	"testing"
)

func TestGetLicenseURL(t *testing.T) {
	tests := []struct {
		name      string
		licenseID string
		expected  string
	}{
		{
			name:      "apache-2.0",
			licenseID: "apache-2.0",
			expected:  "https://www.apache.org/licenses/LICENSE-2.0",
		},
		{
			name:      "MIT license",
			licenseID: "mit",
			expected:  "https://opensource.org/licenses/MIT",
		},
		{
			name:      "case insensitive",
			licenseID: "APACHE-2.0",
			expected:  "https://www.apache.org/licenses/LICENSE-2.0",
		},
		{
			name:      "whitespace handling",
			licenseID: "  apache-2.0  ",
			expected:  "https://www.apache.org/licenses/LICENSE-2.0",
		},
		{
			name:      "unknown license",
			licenseID: "unknown-license",
			expected:  "",
		},
		{
			name:      "llama license",
			licenseID: "llama3.1",
			expected:  "https://github.com/meta-llvm/llama-models/blob/main/models/llama3_1/LICENSE",
		},
		{
			name:      "empty string",
			licenseID: "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLicenseURL(tt.licenseID)
			if result != tt.expected {
				t.Errorf("GetLicenseURL() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
