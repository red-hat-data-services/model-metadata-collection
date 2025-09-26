package catalog

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/opendatahub-io/model-metadata-collection/pkg/types"
)

// LoadStaticCatalogs loads static catalog files and returns their models
func LoadStaticCatalogs(filePaths []string) ([]types.CatalogMetadata, error) {
	var allStaticModels []types.CatalogMetadata

	for _, filePath := range filePaths {
		log.Printf("  Loading static catalog: %s", filePath)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("  Warning: Static catalog file not found: %s", filePath)
			continue
		}

		// Read the file
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("  Error reading static catalog file %s: %v", filePath, err)
			continue
		}

		// Parse the YAML
		var staticCatalog types.ModelsCatalog
		err = yaml.Unmarshal(data, &staticCatalog)
		if err != nil {
			log.Printf("  Error parsing static catalog file %s: %v", filePath, err)
			continue
		}

		// Validate the catalog structure
		if err := validateStaticCatalog(&staticCatalog); err != nil {
			log.Printf("  Error validating static catalog file %s: %v", filePath, err)
			continue
		}

		// Add models from this catalog
		allStaticModels = append(allStaticModels, staticCatalog.Models...)
		log.Printf("  Successfully loaded %d models from %s", len(staticCatalog.Models), filePath)
	}

	log.Printf("Total static models loaded: %d", len(allStaticModels))
	return allStaticModels, nil
}

// validateStaticCatalog validates the structure of a static catalog
func validateStaticCatalog(catalog *types.ModelsCatalog) error {
	if catalog.Source == "" {
		return fmt.Errorf("static catalog missing required 'source' field")
	}

	for i, model := range catalog.Models {
		if model.Name == nil || *model.Name == "" {
			return fmt.Errorf("model at index %d missing required 'name' field", i)
		}

		if len(model.Artifacts) == 0 {
			return fmt.Errorf("model '%s' has no artifacts", *model.Name)
		}

		// Validate each artifact has a URI
		for j, artifact := range model.Artifacts {
			if artifact.URI == "" {
				return fmt.Errorf("model '%s' artifact at index %d missing required 'uri' field", *model.Name, j)
			}
		}
	}

	return nil
}

// CreateModelsCatalogWithStatic collects all metadata.yaml files, merges with static models, and creates a models-catalog.yaml
func CreateModelsCatalogWithStatic(outputDir, catalogPath string, staticModels []types.CatalogMetadata) error {
	var allModels []types.ExtractedMetadata

	// Find all metadata.yaml files in the specified output directory
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
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

	// Convert dynamic models to catalog metadata (excluding tags)
	var catalogModels []types.CatalogMetadata
	for _, model := range allModels {
		catalogModel := convertExtractedToCatalogMetadata(model)
		catalogModels = append(catalogModels, catalogModel)
	}

	// Merge static models with dynamic models (static models are appended at the end)
	catalogModels = append(catalogModels, staticModels...)

	// Create the catalog structure
	catalog := types.ModelsCatalog{
		Source: "Red Hat",
		Models: catalogModels,
	}

	// Marshal to YAML
	output, err := yaml.Marshal(&catalog)
	if err != nil {
		return fmt.Errorf("error marshaling catalog: %v", err)
	}

	// Write to the specified catalog path
	err = os.WriteFile(catalogPath, output, 0644)
	if err != nil {
		return fmt.Errorf("error writing catalog file: %v", err)
	}

	log.Printf("Successfully created %s with %d dynamic models and %d static models", catalogPath, len(allModels), len(staticModels))
	return nil
}

// CreateModelsCatalog collects all metadata.yaml files and creates a models-catalog.yaml (backward compatibility)
func CreateModelsCatalog(outputDir, catalogPath string) error {
	return CreateModelsCatalogWithStatic(outputDir, catalogPath, []types.CatalogMetadata{})
}

// convertExtractedToCatalogMetadata converts ExtractedMetadata to CatalogMetadata
func convertExtractedToCatalogMetadata(model types.ExtractedMetadata) types.CatalogMetadata {
	// Convert timestamps to strings and use artifact values when model values are null
	createTimeStr := convertTimestampToString(model.CreateTimeSinceEpoch)
	lastUpdateTimeStr := convertTimestampToString(model.LastUpdateTimeSinceEpoch)

	// If model timestamps are null, try to use values from the first artifact
	if createTimeStr == nil && len(model.Artifacts) > 0 {
		if model.Artifacts[0].CreateTimeSinceEpoch != nil {
			createTimeStr = convertTimestampToString(model.Artifacts[0].CreateTimeSinceEpoch)
		}
	}
	if lastUpdateTimeStr == nil && len(model.Artifacts) > 0 {
		if model.Artifacts[0].LastUpdateTimeSinceEpoch != nil {
			lastUpdateTimeStr = convertTimestampToString(model.Artifacts[0].LastUpdateTimeSinceEpoch)
		}
	}

	// Convert artifacts to catalog format with string timestamps
	var catalogArtifacts []types.CatalogOCIArtifact
	for _, artifact := range model.Artifacts {
		catalogArtifact := types.CatalogOCIArtifact{
			URI:                      artifact.URI,
			CreateTimeSinceEpoch:     convertTimestampToString(artifact.CreateTimeSinceEpoch),
			LastUpdateTimeSinceEpoch: convertTimestampToString(artifact.LastUpdateTimeSinceEpoch),
			CustomProperties:         convertCustomPropertiesToMetadataValue(artifact.CustomProperties),
		}
		catalogArtifacts = append(catalogArtifacts, catalogArtifact)
	}

	// Convert tags to customProperties
	customProps := convertTagsToCustomProperties(model.Tags)

	// Add validated_on as customProperty if present
	if len(model.ValidatedOn) > 0 {
		validatedOnValue := strings.Join(model.ValidatedOn, ",")
		customProps["validated_on"] = createMetadataValue(validatedOnValue)
	}

	return types.CatalogMetadata{
		Name:                     model.Name,
		Provider:                 model.Provider,
		Description:              model.Description,
		Readme:                   model.Readme,
		Language:                 model.Language,
		License:                  model.License,
		LicenseLink:              model.LicenseLink,
		Tasks:                    model.Tasks,
		CreateTimeSinceEpoch:     createTimeStr,
		LastUpdateTimeSinceEpoch: lastUpdateTimeStr,
		CustomProperties:         customProps,
		Artifacts:                catalogArtifacts,
		Logo:                     determineLogo(model.Tags),
	}
}

// convertTimestampToString converts an int64 timestamp to a string, returning nil if input is nil
func convertTimestampToString(timestamp *int64) *string {
	if timestamp == nil {
		return nil
	}
	str := strconv.FormatInt(*timestamp, 10)
	return &str
}

// convertTagsToCustomProperties converts all tags to customProperties format
func convertTagsToCustomProperties(tags []string) map[string]types.MetadataValue {
	customProps := make(map[string]types.MetadataValue)

	for _, tag := range tags {
		if tag != "" { // Skip empty tags
			customProps[tag] = createMetadataValue("")
		}
	}

	return customProps
}

// createMetadataValue creates a MetadataValue with the standard format
func createMetadataValue(value string) types.MetadataValue {
	return types.MetadataValue{
		MetadataType: "MetadataStringValue",
		StringValue:  value,
	}
}

// convertCustomPropertiesToMetadataValue converts CustomProperties from interface{} to MetadataValue format
func convertCustomPropertiesToMetadataValue(customProps map[string]interface{}) map[string]interface{} {
	if customProps == nil {
		return nil
	}

	result := make(map[string]interface{})
	for key, value := range customProps {
		result[key] = ensureMetadataValueFormat(value)
	}

	return result
}

// ensureMetadataValueFormat ensures a value is in the proper MetadataValue format with metadataType
func ensureMetadataValueFormat(value interface{}) map[string]interface{} {
	// Check if value is already in the correct MetadataValue format
	if valueMap, ok := value.(map[string]interface{}); ok {
		// Check if it already has metadataType
		if _, hasMetadataType := valueMap["metadataType"]; hasMetadataType {
			return valueMap
		} else {
			// Convert to proper MetadataValue format
			stringValue := ""
			if strVal, hasStringValue := valueMap["string_value"]; hasStringValue {
				if str, ok := strVal.(string); ok {
					stringValue = str
				}
			}
			return map[string]interface{}{
				"metadataType": "MetadataStringValue",
				"string_value": stringValue,
			}
		}
	} else {
		// Convert simple values to MetadataValue format
		stringValue := ""
		if str, ok := value.(string); ok {
			stringValue = str
		}
		return map[string]interface{}{
			"metadataType": "MetadataStringValue",
			"string_value": stringValue,
		}
	}
}

// determineLogo determines which logo to use based on model tags and returns base64-encoded data URI
func determineLogo(tags []string) *string {
	var svgPath string

	// Check if the model has the "validated" label
	for _, tag := range tags {
		if tag == "validated" {
			svgPath = "assets/catalog-validated_model.svg"
			break
		}
	}

	// Default logo for non-validated models
	if svgPath == "" {
		svgPath = "assets/catalog-model.svg"
	}

	// Read and encode the SVG file
	dataUri := encodeSVGToDataURI(svgPath)
	return dataUri
}

// encodeSVGToDataURI reads an SVG file and returns a base64-encoded data URI
func encodeSVGToDataURI(svgPath string) *string {
	// Read the SVG file
	svgContent, err := os.ReadFile(svgPath)
	if err != nil {
		log.Printf("Warning: Failed to read SVG file %s: %v", svgPath, err)
		// Return the file path as fallback
		fallback := svgPath
		return &fallback
	}

	// Encode to base64
	base64Content := base64.StdEncoding.EncodeToString(svgContent)

	// Create data URI
	dataUri := "data:image/svg+xml;base64," + base64Content
	return &dataUri
}
