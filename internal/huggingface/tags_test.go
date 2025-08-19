package huggingface

import (
	"reflect"
	"testing"
)

func TestParseTagsForStructuredData(t *testing.T) {
	tests := []struct {
		name              string
		tags              []string
		expectedLanguages []string
		expectedLicense   string
		expectedTasks     []string
	}{
		{
			name:              "basic language and task tags",
			tags:              []string{"en", "text-generation", "conversational"},
			expectedLanguages: []string{"en"},
			expectedLicense:   "",
			expectedTasks:     []string{"text-generation"},
		},
		{
			name:              "license tag",
			tags:              []string{"license:apache-2.0", "en", "text-generation"},
			expectedLanguages: []string{"en"},
			expectedLicense:   "apache-2.0",
			expectedTasks:     []string{"text-generation"},
		},
		{
			name:              "multiple languages",
			tags:              []string{"en", "es", "fr", "text-classification"},
			expectedLanguages: []string{"en", "es", "fr"},
			expectedLicense:   "",
			expectedTasks:     []string{"text-classification"},
		},
		{
			name:              "tools and libraries filtered out",
			tags:              []string{"pytorch", "vllm", "transformers", "text-generation"},
			expectedLanguages: nil,
			expectedLicense:   "",
			expectedTasks:     []string{"text-generation"},
		},
		{
			name:              "unknown tags ignored",
			tags:              []string{"unknown-tag", "custom-tag", "en", "text-generation"},
			expectedLanguages: []string{"en"},
			expectedLicense:   "",
			expectedTasks:     []string{"text-generation"},
		},
		{
			name:              "empty tags",
			tags:              []string{},
			expectedLanguages: nil,
			expectedLicense:   "",
			expectedTasks:     nil,
		},
		{
			name:              "case insensitive processing",
			tags:              []string{"EN", "TEXT-GENERATION", "LICENSE:MIT"},
			expectedLanguages: []string{"en"},
			expectedLicense:   "mit",
			expectedTasks:     []string{"text-generation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			languages, license, tasks := ParseTagsForStructuredData(tt.tags)

			if !reflect.DeepEqual(languages, tt.expectedLanguages) {
				t.Errorf("ParseTagsForStructuredData() languages = %v, expected %v", languages, tt.expectedLanguages)
			}

			if license != tt.expectedLicense {
				t.Errorf("ParseTagsForStructuredData() license = %q, expected %q", license, tt.expectedLicense)
			}

			if !reflect.DeepEqual(tasks, tt.expectedTasks) {
				t.Errorf("ParseTagsForStructuredData() tasks = %v, expected %v", tasks, tt.expectedTasks)
			}
		})
	}
}
