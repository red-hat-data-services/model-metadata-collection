package metadata

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/pkg/types"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/pkg/utils"
)

// ModelCardYAMLFrontmatter represents the YAML frontmatter in modelcard.md files
type ModelCardYAMLFrontmatter struct {
	LibraryName string   `yaml:"library_name"`
	Language    []string `yaml:"language"`
	PipelineTag string   `yaml:"pipeline_tag"`
	License     string   `yaml:"license"`
	LicenseName string   `yaml:"license_name"`
	Tags        []string `yaml:"tags"`
}

// ExtractYAMLFrontmatterFromModelCard extracts YAML frontmatter from modelcard.md content
func ExtractYAMLFrontmatterFromModelCard(content string) (*ModelCardYAMLFrontmatter, error) {
	if content == "" {
		return nil, fmt.Errorf("empty modelcard content")
	}

	// Check if content starts with YAML frontmatter (---)
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("no YAML frontmatter found")
	}

	// Find the end of the frontmatter
	lines := strings.Split(content, "\n")
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		return nil, fmt.Errorf("malformed YAML frontmatter: no closing ---")
	}

	// Extract and parse YAML content
	yamlContent := strings.Join(lines[1:endIndex], "\n")
	var frontmatter ModelCardYAMLFrontmatter
	err := yaml.Unmarshal([]byte(yamlContent), &frontmatter)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %v", err)
	}

	return &frontmatter, nil
}

// splitTaskString intelligently splits task strings while preserving URLs and markdown links
func splitTaskString(taskStr string) []string {
	// First, extract meaningful task-like terms before trying to split
	// Look for common task patterns and known task types
	var tasks []string

	// Check for common task patterns
	taskPatterns := []string{
		`text-generation`,
		`text-to-text-generation`,
		`conversational`,
		`question-answering`,
		`summarization`,
		`translation`,
		`code-generation`,
		`chat`,
		`instruction-following`,
		`assistant-like chat`,
		`commercial and research use`,
	}

	taskStr = strings.ToLower(taskStr)
	for _, pattern := range taskPatterns {
		if strings.Contains(taskStr, pattern) {
			tasks = append(tasks, pattern)
		}
	}

	// If we found task-like patterns, return them, otherwise try simple comma splitting
	if len(tasks) > 0 {
		return tasks
	}

	// Fallback: split only on commas and semicolons, avoid periods which break URLs
	return strings.FieldsFunc(taskStr, func(c rune) bool {
		return c == ',' || c == ';'
	})
}

// parseModelCardMetadata extracts metadata presence from modelcard markdown content
func ParseModelCardMetadata(content []byte) types.ModelMetadata {
	contentStr := strings.ToLower(string(content))

	return types.ModelMetadata{
		Name:                     utils.ContainsMetadataField(contentStr, []string{"name:", "model name", "# "}),
		Provider:                 utils.ContainsMetadataField(contentStr, []string{"provider:", "model developers:", "developers:", "author"}),
		Description:              utils.ContainsMetadataField(contentStr, []string{"description:", "## model overview", "overview"}),
		Readme:                   len(content) > 0, // If we have content, we have a readme
		Language:                 utils.ContainsMetadataField(contentStr, []string{"language:", "languages:", "supported language"}),
		License:                  utils.ContainsMetadataField(contentStr, []string{"license:", "licensing", "apache", "mit", "gpl"}),
		LicenseLink:              utils.ContainsMetadataField(contentStr, []string{"license link", "license url", "licensing terms", "www.apache.org"}),
		Maturity:                 utils.ContainsMetadataField(contentStr, []string{"maturity:", "development", "production", "beta", "alpha", "stable"}),
		LibraryName:              utils.ContainsMetadataField(contentStr, []string{"library:", "framework:", "vllm", "transformers", "pytorch", "tensorflow"}),
		Tasks:                    utils.ContainsMetadataField(contentStr, []string{"task:", "tasks:", "use case", "application"}),
		CreateTimeSinceEpoch:     utils.ContainsMetadataField(contentStr, []string{"created:", "creation date", "release date:", "date:"}),
		LastUpdateTimeSinceEpoch: utils.ContainsMetadataField(contentStr, []string{"updated:", "last update", "modified:", "version:"}),
		Artifacts:                utils.ContainsMetadataField(contentStr, []string{"artifact:", "model file", "download", "huggingface", "registry.redhat.io"}),
	}
}

// ExtractMetadataValues extracts actual values from modelcard markdown content with validation
func ExtractMetadataValues(content []byte) types.ExtractedMetadata {
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	metadata := types.ExtractedMetadata{}

	// First, try to extract YAML frontmatter
	frontmatter, err := ExtractYAMLFrontmatterFromModelCard(contentStr)
	if err == nil {
		// Populate metadata from YAML frontmatter first (highest priority)

		// Library name from YAML
		if frontmatter.LibraryName != "" {
			metadata.LibraryName = &frontmatter.LibraryName
		}

		// Language from YAML
		if len(frontmatter.Language) > 0 {
			metadata.Language = frontmatter.Language
		}

		// License from YAML (prefer license_name if available)
		if frontmatter.LicenseName != "" {
			metadata.License = &frontmatter.LicenseName
			// Automatically set license link if we have a well-known license
			if licenseURL := utils.GetLicenseURL(frontmatter.LicenseName); licenseURL != "" {
				metadata.LicenseLink = &licenseURL
			}
		} else if frontmatter.License != "" {
			metadata.License = &frontmatter.License
			if licenseURL := utils.GetLicenseURL(frontmatter.License); licenseURL != "" {
				metadata.LicenseLink = &licenseURL
			}
		}

		// Pipeline tag for tasks
		if frontmatter.PipelineTag != "" {
			metadata.Tasks = []string{frontmatter.PipelineTag}
		}
	}

	// Extract name from title - look for model-like headings, not code examples
	titleRegex := regexp.MustCompile(`(?m)^#\s+(.+)$`)
	titleMatches := titleRegex.FindAllStringSubmatch(contentStr, -1)

	for _, titleMatch := range titleMatches {
		name := utils.CleanExtractedValue(titleMatch[1])
		// Skip obvious code examples, function definitions, or generic headings
		if strings.Contains(strings.ToLower(name), "define") ||
			strings.Contains(strings.ToLower(name), "function") ||
			strings.Contains(strings.ToLower(name), "tool") ||
			strings.Contains(strings.ToLower(name), "example") ||
			strings.Contains(name, "(") ||
			strings.Contains(name, "def ") {
			continue
		}
		// Look for model name patterns (contains model-like terms or version numbers)
		if (strings.Contains(strings.ToLower(name), "model") ||
			strings.Contains(strings.ToLower(name), "llama") ||
			strings.Contains(strings.ToLower(name), "granite") ||
			strings.Contains(strings.ToLower(name), "mistral") ||
			strings.Contains(strings.ToLower(name), "qwen") ||
			strings.Contains(strings.ToLower(name), "phi") ||
			regexp.MustCompile(`\d+[.-]\d+`).MatchString(name) ||
			regexp.MustCompile(`(?i)(instruct|base|quantized|fp8|w\d+a\d+)`).MatchString(name)) &&
			utils.IsValidValue(name, 3, 100, nil) {
			metadata.Name = &name
			break
		}
	}

	// Extract provider/developers from structured fields - enhanced patterns
	for _, line := range lines {
		// Try multiple provider patterns
		patterns := []string{
			`(?i)^-?\s*\*?\*?(?:Model Developers?|Developers?|Author|Provider|Authors?):\*?\*?\s*(.+)$`,
			`(?i)^-?\s*\*?\*?(?:Developed by|Created by|Made by):\*?\*?\s*(.+)$`,
			`(?i)^-?\s*\*?\*?(?:Company|Organization|Team):\*?\*?\s*(.+)$`,
		}

		for _, pattern := range patterns {
			if providerMatch := regexp.MustCompile(pattern).FindStringSubmatch(line); providerMatch != nil {
				provider := utils.CleanExtractedValue(providerMatch[1])
				// More lenient validation for provider names
				if utils.IsValidValue(provider, 2, 100, []string{`^[A-Za-z0-9\s\\.&,\-()]+$`}) {
					metadata.Provider = &provider
					break
				}
			}
		}
		if metadata.Provider != nil {
			break
		}
	}

	// Additional provider extraction from model cards that mention well-known companies
	if metadata.Provider == nil {
		// Look for company mentions in first few paragraphs
		companyRegex := regexp.MustCompile(`(?i)(IBM|Microsoft|Meta|Google|OpenAI|Anthropic|Mistral|Neural Magic|Red Hat|Hugging Face|Facebook)\s+(?:Research|AI|Inc\.?|Corporation|Corp\.?)?`)
		if companyMatch := companyRegex.FindStringSubmatch(contentStr); companyMatch != nil {
			company := strings.TrimSpace(companyMatch[0])
			metadata.Provider = &company
		}
	}

	// Extract description from Model Overview or first paragraph after title
	overviewRegex := regexp.MustCompile(`(?i)(?:## Model Overview|## Overview)\s*\n((?:[^\n]+\n)*?)(?:\n##|\n#|$)`)
	if overviewMatch := overviewRegex.FindStringSubmatch(contentStr); overviewMatch != nil {
		// Look for description in overview section
		overviewText := overviewMatch[1]
		if descMatch := regexp.MustCompile(`(?i)(?:^|\n)\s*(.+?(?:model|quantized version|intended for).{20,200}?)(?:\n|$)`).FindStringSubmatch(overviewText); descMatch != nil {
			desc := utils.CleanExtractedValue(descMatch[1])
			if utils.IsValidValue(desc, 20, 500, nil) {
				metadata.Description = &desc
			}
		}
	}

	// Fallback: first paragraph after title
	if metadata.Description == nil {
		descRegex := regexp.MustCompile(`(?s)^#[^\n]+\n\n([^\n#]+(?:\n[^\n#]+)*?)(?:\n\n|\n#|$)`)
		if descMatch := descRegex.FindStringSubmatch(contentStr); descMatch != nil {
			desc := utils.CleanExtractedValue(descMatch[1])
			if utils.IsValidValue(desc, 20, 500, nil) {
				metadata.Description = &desc
			}
		}
	}

	// Final description fallback: generate from model name if still none
	if metadata.Description == nil && metadata.Name != nil {
		fallbackDesc := utils.GenerateReadableDescription(*metadata.Name)
		if fallbackDesc != "" {
			metadata.Description = &fallbackDesc
		}
	}

	// Readme is the full content
	if len(content) > 0 {
		readme := string(content)
		metadata.Readme = &readme
	}

	// Extract license from structured fields (only if not already set by YAML frontmatter)
	if metadata.License == nil {
		for _, line := range lines {
			if licenseMatch := regexp.MustCompile(`(?i)^-?\s*\*?\*?(?:License(?:\(s\))?|Licensing):\*?\*?\s*(?:\[([^\]]+)\]|\*?([A-Za-z0-9\.\-_]+)\*?)`).FindStringSubmatch(line); licenseMatch != nil {
				var license string
				if licenseMatch[1] != "" {
					license = utils.CleanExtractedValue(licenseMatch[1])
				} else {
					license = utils.CleanExtractedValue(licenseMatch[2])
				}
				if utils.IsValidValue(license, 2, 30, []string{`^[A-Za-z0-9\.\-_\s]+$`}) {
					metadata.License = &license
					// Automatically set license link if we have a well-known license
					if licenseURL := utils.GetLicenseURL(license); licenseURL != "" {
						metadata.LicenseLink = &licenseURL
					}
					break
				}
			}
		}
	}

	// Extract license link (only if not already set by YAML frontmatter)
	if metadata.LicenseLink == nil {
		licenseLinkRegex := regexp.MustCompile(`(?i)(?:license|licensing)[^\(]*\((https?://[^\)]+)\)`)
		if linkMatch := licenseLinkRegex.FindStringSubmatch(contentStr); linkMatch != nil {
			link := strings.TrimSpace(linkMatch[1])
			if utils.IsValidValue(link, 10, 200, []string{`^https?://`}) {
				metadata.LicenseLink = &link
			}
		}
	}

	// Extract release date from structured fields and convert to epoch
	for _, line := range lines {
		if dateMatch := regexp.MustCompile(`(?i)^-?\s*\*?\*?(?:Release Date|Date):\*?\*?\s*([0-9]{1,2}[\/\-][0-9]{1,2}[\/\-][0-9]{4})`).FindStringSubmatch(line); dateMatch != nil {
			if epoch := utils.ParseDateToEpoch(dateMatch[1]); epoch != nil {
				metadata.CreateTimeSinceEpoch = epoch
				break
			}
		}
	}

	// Extract version from structured fields and convert version date to epoch if possible
	for _, line := range lines {
		if versionMatch := regexp.MustCompile(`(?i)^-?\s*\*?\*?Version:\*?\*?\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`).FindStringSubmatch(line); versionMatch != nil {
			// For version numbers, we'll look for any associated date in the same section
			// If no date is found associated with version, we'll leave it null
			// This is because version numbers alone don't represent epoch timestamps
			break
		}
	}

	// Look for any update/modification dates in the content
	updateDateRegex := regexp.MustCompile(`(?i)(?:updated?|modified|last\s+update).*?([0-9]{1,2}[\/\-][0-9]{1,2}[\/\-][0-9]{4})`)
	if updateMatch := updateDateRegex.FindStringSubmatch(contentStr); updateMatch != nil {
		if epoch := utils.ParseDateToEpoch(updateMatch[1]); epoch != nil {
			metadata.LastUpdateTimeSinceEpoch = epoch
		}
	}

	// Extract intended use cases/tasks from structured fields (only if not already set by YAML frontmatter)
	if len(metadata.Tasks) == 0 {
		for _, line := range lines {
			if taskMatch := regexp.MustCompile(`(?i)^-?\s*\*?\*?(?:Intended Use Cases?|Tasks?):\*?\*?\s*(.+)$`).FindStringSubmatch(line); taskMatch != nil {
				taskStr := utils.CleanExtractedValue(taskMatch[1])
				if utils.IsValidValue(taskStr, 5, 200, nil) {
					// Use smarter splitting that preserves URLs and markdown links
					tasks := splitTaskString(taskStr)
					for _, task := range tasks {
						task = utils.CleanExtractedValue(task)
						if utils.IsValidValue(task, 3, 50, nil) {
							// Normalize task to standard format
							normalizedTask := utils.NormalizeTask(task)
							if normalizedTask != "" {
								metadata.Tasks = append(metadata.Tasks, normalizedTask)
							}
						}
					}
					break
				}
			}
		}
	}

	// Extract library/framework - look for vLLM, transformers, etc. in usage context (only if not already set by YAML frontmatter)
	if metadata.LibraryName == nil {
		libRegex := regexp.MustCompile(`(?i)using the \[([a-zA-Z]+)\]|with ([a-zA-Z]+) >=|backend.*?([a-zA-Z]+)`)
		if libMatch := libRegex.FindStringSubmatch(contentStr); libMatch != nil {
			for i := 1; i < len(libMatch); i++ {
				if libMatch[i] != "" {
					lib := utils.CleanExtractedValue(libMatch[i])
					if utils.IsValidValue(lib, 3, 20, []string{`^[a-zA-Z]+$`}) &&
						(strings.ToLower(lib) == "vllm" || strings.ToLower(lib) == "transformers" ||
							strings.ToLower(lib) == "pytorch" || strings.ToLower(lib) == "tensorflow") {
						metadata.LibraryName = &lib
						break
					}
				}
			}
		}
	}

	// Extract language from supported languages sections (only if not already set by YAML frontmatter)
	if len(metadata.Language) == 0 {
		supportedLangsRegex := regexp.MustCompile(`(?i)(?:(?:supported\s+languages?|languages?\s+supported):\s*([^.\n]+)|supports\s+\d+\s+languages?\s+in\s+addition\s+to\s+English:\s*([^.]+))`)
		if langMatch := supportedLangsRegex.FindStringSubmatch(contentStr); langMatch != nil {
			var langStr string
			if langMatch[1] != "" {
				langStr = utils.CleanExtractedValue(langMatch[1])
			} else {
				langStr = "English, " + utils.CleanExtractedValue(langMatch[2]) // Add English since it's mentioned as "in addition to English"
			}
			languages := utils.ParseLanguageNames(langStr)
			if len(languages) > 0 {
				metadata.Language = languages
			}
		} else {
			// Fallback: Extract language from other structured fields
			for _, line := range lines {
				if langMatch := regexp.MustCompile(`(?i)(?:language|languages?).*?(?:in\s+)?([A-Z][a-z]+(?:\s+and\s+[A-Z][a-z]+)*)`).FindStringSubmatch(line); langMatch != nil {
					langStr := utils.CleanExtractedValue(langMatch[1])
					languages := utils.ParseLanguageNames(langStr)
					if len(languages) > 0 {
						metadata.Language = languages
						break
					}
				}
			}
		}
	}

	// Extract OCI image artifacts and model references
	// For now, we'll extract from content but we'll populate with registry data later
	metadata.Artifacts = []types.OCIArtifact{}

	// If lastUpdateTimeSinceEpoch is null but createTimeSinceEpoch has a value,
	// set lastUpdateTimeSinceEpoch to the same value as createTimeSinceEpoch
	if metadata.LastUpdateTimeSinceEpoch == nil && metadata.CreateTimeSinceEpoch != nil {
		lastUpdate := *metadata.CreateTimeSinceEpoch
		metadata.LastUpdateTimeSinceEpoch = &lastUpdate
	}

	return metadata
}
