package catalog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/chambridge/model-metadata-collection/pkg/types"
)

// CreateModelsCatalog collects all metadata.yaml files and creates a models-catalog.yaml
func CreateModelsCatalog() error {
	var allModels []types.ExtractedMetadata

	// Find all metadata.yaml files in output directory
	err := filepath.Walk("output", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == "metadata.yaml" {
			log.Printf("  Processing: %s", path)

			// Read the metadata file
			data, err := os.ReadFile(path)
			if err != nil {
				log.Printf("  Error reading %s: %v", path, err)
				return nil // Continue with other files
			}

			// Parse the YAML
			var metadata types.ExtractedMetadata
			err = yaml.Unmarshal(data, &metadata)
			if err != nil {
				log.Printf("  Error parsing %s: %v", path, err)
				return nil // Continue with other files
			}

			// Add to collection
			allModels = append(allModels, metadata)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %v", err)
	}

	// Sort models by name for consistent output
	sort.Slice(allModels, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		if allModels[i].Name != nil {
			nameI = *allModels[i].Name
		}
		if allModels[j].Name != nil {
			nameJ = *allModels[j].Name
		}
		return nameI < nameJ
	})

	// Create the catalog structure
	catalog := types.ModelsCatalog{
		Source: "Red Hat",
		Models: allModels,
	}

	// Marshal to YAML
	output, err := yaml.Marshal(&catalog)
	if err != nil {
		return fmt.Errorf("error marshaling catalog: %v", err)
	}

	// Write to data/models-catalog.yaml
	catalogPath := "data/models-catalog.yaml"
	err = os.WriteFile(catalogPath, output, 0644)
	if err != nil {
		return fmt.Errorf("error writing catalog file: %v", err)
	}

	log.Printf("Successfully created %s with %d models", catalogPath, len(allModels))
	return nil
}
