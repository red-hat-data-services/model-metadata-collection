package utils

import (
	"regexp"
	"strings"
	"time"
)

// isValidValue checks if a value meets basic quality criteria
func IsValidValue(value string, minLength int, maxLength int, allowedPatterns []string) bool {
	if len(value) < minLength || len(value) > maxLength {
		return false
	}

	// Check if value contains only reasonable characters (no control chars, etc.)
	for _, r := range value {
		if r < 32 || r > 126 {
			return false
		}
	}

	// If patterns are specified, value must match at least one
	if len(allowedPatterns) > 0 {
		for _, pattern := range allowedPatterns {
			if matched, _ := regexp.MatchString(pattern, value); matched {
				return true
			}
		}
		return false
	}

	return true
}

// cleanExtractedValue removes common artifacts and validates the value
func CleanExtractedValue(value string) string {
	// Remove markdown formatting
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "*_`\"")

	// Remove trailing colons and periods
	value = strings.TrimRight(value, ":.")

	return value
}

// containsMetadataField checks if any of the field indicators are present in the content
func ContainsMetadataField(content string, indicators []string) bool {
	for _, indicator := range indicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	return false
}

// sanitizeManifestRef creates a valid directory name from manifestRef
func SanitizeManifestRef(manifestRef string) string {
	// Replace invalid filesystem characters with underscores
	// Invalid characters: / \ : * ? " < > |
	re := regexp.MustCompile(`[/\\:*?"<>|]`)
	sanitized := re.ReplaceAllString(manifestRef, "_")

	// Replace multiple consecutive underscores with a single one
	re = regexp.MustCompile(`_+`)
	sanitized = re.ReplaceAllString(sanitized, "_")

	// Remove leading/trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	return sanitized
}

// parseDateToEpoch converts a date string to Unix epoch timestamp in milliseconds
func ParseDateToEpoch(dateStr string) *int64 {
	dateStr = CleanExtractedValue(dateStr)

	// Try various date formats
	formats := []string{
		"1/2/2006",   // M/D/YYYY
		"01/02/2006", // MM/DD/YYYY
		"1-2-2006",   // M-D-YYYY
		"01-02-2006", // MM-DD-YYYY
		"2006-01-02", // YYYY-MM-DD
		"2/1/2006",   // D/M/YYYY
		"02/01/2006", // DD/MM/YYYY
		"2-1-2006",   // D-M-YYYY
		"02-01-2006", // DD-MM-YYYY
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			epoch := t.Unix() * 1000
			return &epoch
		}
	}

	return nil
}

// parseTimeToEpochInt64 converts ISO 8601 timestamp to epoch milliseconds int64
func ParseTimeToEpochInt64(timeStr string) *int64 {
	if timeStr == "" {
		return nil
	}

	// Try various time formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			epoch := t.Unix() * 1000
			return &epoch
		}
	}

	return nil
}
