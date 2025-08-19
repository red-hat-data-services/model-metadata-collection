package huggingface

import (
	"regexp"
	"strings"
)

// ParseTagsForStructuredData extracts structured metadata from HuggingFace tags
func ParseTagsForStructuredData(tags []string) (languages []string, license string, tasks []string) {
	// Known language codes
	languageCodes := map[string]bool{
		"en": true, "fr": true, "de": true, "es": true, "it": true, "pt": true, "ru": true,
		"zh": true, "ja": true, "ko": true, "ar": true, "hi": true, "nl": true, "sv": true,
		"da": true, "no": true, "fi": true, "pl": true, "cs": true, "hu": true, "tr": true,
		"he": true, "th": true, "vi": true, "id": true, "ms": true, "tl": true, "sw": true,
	}

	// Known task types mapped to standardized format
	taskTypes := map[string]string{
		"text-generation":              "text-generation",
		"text-classification":          "text-classification",
		"text-to-text-generation":      "text-generation",
		"translation":                  "text-generation",
		"summarization":                "text-generation",
		"question-answering":           "question-answering",
		"conversational":               "text-generation",
		"text-to-speech":               "text-generation",
		"automatic-speech-recognition": "text-generation",
		"image-classification":         "image-classification",
		"image-to-text":                "image-to-text",
		"text-to-image":                "text-generation",
		"feature-extraction":           "text-generation",
		"sentence-similarity":          "sentence-similarity",
		"zero-shot-classification":     "text-classification",
		"token-classification":         "text-classification",
		"fill-mask":                    "text-generation",
		"multiple-choice":              "question-answering",
		"table-question-answering":     "question-answering",
		"visual-question-answering":    "question-answering",
		"any-to-any":                   "any-to-any",
		"image-text-to-text":           "image-text-to-text",
		"image-to-image":               "image-to-image",
		"text-ranking":                 "text-ranking",
		"text-to-video":                "text-to-video",
		"video-to-video":               "video-to-video",
	}

	// Known license patterns (for detecting license names in tags)
	licensePatterns := map[string]string{
		"llama2":       "llama2",
		"llama3":       "llama3",
		"llama3.1":     "llama3.1",
		"llama3.2":     "llama3.2",
		"llama3.3":     "llama3.3",
		"llama4":       "llama4",
		"apache-2.0":   "apache-2.0",
		"mit":          "mit",
		"gpl-3.0":      "gpl-3.0",
		"bsd-3-clause": "bsd-3-clause",
	}

	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))

		// Extract license from license: prefix
		if strings.HasPrefix(tag, "license:") {
			extractedLicense := strings.TrimPrefix(tag, "license:")
			// Only use it if it's not "other" - we'll try to find a better license below
			if extractedLicense != "other" {
				license = extractedLicense
			}
			continue
		}

		// Check if the tag itself is a license name (for cases where license:other but llama4 tag exists)
		if licenseValue, isLicense := licensePatterns[tag]; isLicense {
			// Prefer specific license names over "other"
			if license == "" || license == "other" {
				license = licenseValue
			}
			continue
		}

		// Check if it's a language code
		if languageCodes[tag] {
			languages = append(languages, tag)
			continue
		}

		// Check if it's a task type and normalize it
		if normalizedTask, exists := taskTypes[tag]; exists {
			// Avoid duplicates
			found := false
			for _, existingTask := range tasks {
				if existingTask == normalizedTask {
					found = true
					break
				}
			}
			if !found {
				tasks = append(tasks, normalizedTask)
			}
			continue
		}
	}

	return languages, license, tasks
}

// InferTasksFromReadme attempts to infer model tasks from README content
func InferTasksFromReadme(readme string) []string {
	var tasks []string

	// Pattern to detect text input/output architecture - flexible patterns
	patterns := []string{
		// Pattern 1: **Input:** Text ... **Output:** Text (with potential content in between)
		`(?i)\*\*Input:\*\*\s*Text.*?\*\*Output:\*\*\s*Text`,
		// Pattern 2: - **Input:** Text and - **Output:** Text (separate lines)
		`(?i)-\s*\*\*Input:\*\*\s*Text[\s\S]*?-\s*\*\*Output:\*\*\s*Text`,
		// Pattern 3: Input: Text and Output: Text (without asterisks)
		`(?i)Input:\s*Text[\s\S]*?Output:\s*Text`,
	}

	for _, pattern := range patterns {
		textIOPattern := regexp.MustCompile(pattern)
		if textIOPattern.MatchString(readme) {
			tasks = append(tasks, "text-generation")
			break // Found one pattern, no need to check others
		}
	}

	return tasks
}
