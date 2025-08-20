package huggingface

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/pkg/types"
)

// parseVersionFromTitle extracts version from collection title using semver patterns
func parseVersionFromTitle(title string) string {
	// Look for version patterns like "v1.0", "v2.1", "v1.0.0", etc.
	versionRegex := regexp.MustCompile(`v(\d+\.\d+(?:\.\d+)?)`)
	matches := versionRegex.FindStringSubmatch(strings.ToLower(title))
	if len(matches) > 1 {
		return "v" + matches[1]
	}
	return ""
}

// generateVersionIndex creates an index file for a specific version
func generateVersionIndex(collection *types.HFCollection, version string) error {
	var models []types.ModelIndex

	for _, model := range collection.Items {
		// Create model index entry
		modelIndex := types.ModelIndex{
			Name:       model.ID,
			URL:        fmt.Sprintf("https://huggingface.co/%s", model.ID),
			ReadmePath: fmt.Sprintf("/%s/README.md", model.ID),
		}
		models = append(models, modelIndex)
	}

	versionIndex := types.VersionIndex{
		Version: version,
		Models:  models,
	}

	// Ensure data directory exists
	err := os.MkdirAll("data", 0755)
	if err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Generate filename
	filename := fmt.Sprintf("data/hugging-face-redhat-ai-validated-%s.yaml", strings.ReplaceAll(version, ".", "-"))

	// Marshal to YAML
	yamlData, err := yaml.Marshal(versionIndex)
	if err != nil {
		return fmt.Errorf("failed to marshal version index to YAML: %v", err)
	}

	// Write to file
	err = os.WriteFile(filename, yamlData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write version index file: %v", err)
	}

	log.Printf("Generated index file: %s with %d models", filename, len(models))
	return nil
}

// ProcessCollections processes all HuggingFace collections and generates index files
func ProcessCollections() error {
	log.Println("Discovering Red Hat AI validated model collections...")

	// Try to discover collections automatically
	collectionSlugs, err := DiscoverValidatedModelCollections()
	if err != nil {
		log.Printf("Failed to discover collections, using known collection: %v", err)
		// Fall back to known collection
		collectionSlugs = []string{"RedHatAI/red-hat-ai-validated-models-v10-682613dc19c4a596dbac9437"}
	}

	if len(collectionSlugs) == 0 {
		return fmt.Errorf("no validated model collections found")
	}

	// Process each discovered collection
	for _, slug := range collectionSlugs {
		log.Printf("Processing collection: %s", slug)

		collection, err := FetchCollectionDetails(slug)
		if err != nil {
			log.Printf("Failed to fetch collection details for %s: %v", slug, err)
			continue
		}

		log.Printf("Found collection: %s", collection.Title)

		// Parse version from title
		version := parseVersionFromTitle(collection.Title)
		if version == "" {
			version = "v1.0" // Default fallback
		}

		log.Printf("Detected version: %s", version)

		// Generate index file for this version
		err = generateVersionIndex(collection, version)
		if err != nil {
			log.Printf("Failed to generate version index for %s: %v", version, err)
			continue
		}
	}

	return nil
}
