package enrichment

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/huggingface"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/metadata"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/pkg/types"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/pkg/utils"
)

// UpdateModelMetadataFile updates an existing metadata.yaml file with enriched data and creates separate enrichment.yaml
func UpdateModelMetadataFile(registryModel string, enrichedData *types.EnrichedModelMetadata) error {
	// Create sanitized directory name for the model
	sanitizedName := utils.SanitizeManifestRef(registryModel)
	metadataPath := fmt.Sprintf("output/%s/models/metadata.yaml", sanitizedName)
	enrichmentPath := fmt.Sprintf("output/%s/models/enrichment.yaml", sanitizedName)

	// Try to load existing metadata using migration logic
	existingMetadataPtr, err := metadata.LoadExistingMetadata(registryModel)
	var existingMetadata types.ExtractedMetadata
	if err == nil && existingMetadataPtr != nil {
		existingMetadata = *existingMetadataPtr
	} else {
		// If loading fails, this could mean the metadata file doesn't exist yet
		// In this case, we start with an empty struct, which is correct
		log.Printf("  Warning: Could not load existing metadata for %s: %v", registryModel, err)
	}

	// Create enrichment structure with granular source tracking
	enrichmentInfo := struct {
		HuggingFaceModel string `yaml:"huggingface_model,omitempty"`
		HuggingFaceURL   string `yaml:"huggingface_url,omitempty"`
		MatchConfidence  string `yaml:"match_confidence,omitempty"`
		DataSources      struct {
			Name                 string `yaml:"name,omitempty"`
			Provider             string `yaml:"provider,omitempty"`
			Description          string `yaml:"description,omitempty"`
			License              string `yaml:"license,omitempty"`
			LicenseLink          string `yaml:"license_link,omitempty"`
			LibraryName          string `yaml:"library_name,omitempty"`
			Language             string `yaml:"language,omitempty"`
			Tasks                string `yaml:"tasks,omitempty"`
			LastModified         string `yaml:"last_modified,omitempty"`
			CreateTimeSinceEpoch string `yaml:"create_time_since_epoch,omitempty"`
			Readme               string `yaml:"readme,omitempty"`
			Maturity             string `yaml:"maturity,omitempty"`
		} `yaml:"data_sources"`
	}{}

	// Set enrichment info
	enrichmentInfo.HuggingFaceModel = enrichedData.HuggingFaceModel
	enrichmentInfo.HuggingFaceURL = enrichedData.HuggingFaceURL
	enrichmentInfo.MatchConfidence = enrichedData.MatchConfidence

	// Update metadata with enriched values and track sources in enrichment file
	if enrichedData.Name.Source != "null" {
		if existingMetadata.Name == nil {
			nameStr := enrichedData.Name.Value.(string)
			existingMetadata.Name = &nameStr
		}
		enrichmentInfo.DataSources.Name = enrichedData.Name.Source
	}

	if enrichedData.Provider.Source != "null" {
		if existingMetadata.Provider == nil {
			providerStr := enrichedData.Provider.Value.(string)
			existingMetadata.Provider = &providerStr
		}
		enrichmentInfo.DataSources.Provider = enrichedData.Provider.Source
	}

	if enrichedData.Description.Source != "null" {
		if existingMetadata.Description == nil {
			descStr := enrichedData.Description.Value.(string)
			existingMetadata.Description = &descStr
		}
		enrichmentInfo.DataSources.Description = enrichedData.Description.Source
	}

	if enrichedData.License.Source != "null" {
		if existingMetadata.License == nil {
			licenseStr := enrichedData.License.Value.(string)
			existingMetadata.License = &licenseStr
			// Automatically set license link if we have a well-known license
			if licenseURL := utils.GetLicenseURL(licenseStr); licenseURL != "" {
				existingMetadata.LicenseLink = &licenseURL
				enrichmentInfo.DataSources.LicenseLink = "generated"
			}
		}
		enrichmentInfo.DataSources.License = enrichedData.License.Source
	}

	// Handle library name from enriched data
	if enrichedData.LibraryName.Source != "null" {
		if existingMetadata.LibraryName == nil {
			libraryStr := enrichedData.LibraryName.Value.(string)
			existingMetadata.LibraryName = &libraryStr
		}
		enrichmentInfo.DataSources.LibraryName = enrichedData.LibraryName.Source
	}

	// Handle license from tags
	if enrichedData.Tags.Source == "huggingface.tags" && enrichedData.Tags.Value != nil {
		tags, ok := enrichedData.Tags.Value.([]string)
		if ok {
			_, tagLicense, _ := huggingface.ParseTagsForStructuredData(tags)
			if tagLicense != "" && existingMetadata.License == nil {
				existingMetadata.License = &tagLicense
				enrichmentInfo.DataSources.License = "huggingface.tags"
				// Automatically set license link if we have a well-known license
				if licenseURL := utils.GetLicenseURL(tagLicense); licenseURL != "" {
					existingMetadata.LicenseLink = &licenseURL
					enrichmentInfo.DataSources.LicenseLink = "generated"
				}
			}
		}
	}

	// Handle languages from tags
	if enrichedData.Tags.Source == "huggingface.tags" && enrichedData.Tags.Value != nil {
		tags, ok := enrichedData.Tags.Value.([]string)
		if ok {
			languages, _, _ := huggingface.ParseTagsForStructuredData(tags)
			if len(languages) > 0 && len(existingMetadata.Language) == 0 {
				existingMetadata.Language = languages
				enrichmentInfo.DataSources.Language = "huggingface.tags"
			}
		}
	}

	// Handle tasks from enriched data first (highest priority)
	if enrichedData.Tasks.Source != "null" && enrichedData.Tasks.Value != nil {
		tasks, ok := enrichedData.Tasks.Value.([]string)
		if ok && len(tasks) > 0 {
			log.Printf("  Debug: Using tasks from enrichedData.Tasks: %v", tasks)
			existingMetadata.Tasks = tasks
			enrichmentInfo.DataSources.Tasks = enrichedData.Tasks.Source
		}
	} else if enrichedData.Tags.Source == "huggingface.tags" && enrichedData.Tags.Value != nil {
		// Fallback: parse tasks from tags if tasks field is not available
		tags, ok := enrichedData.Tags.Value.([]string)
		if ok {
			_, _, tasks := huggingface.ParseTagsForStructuredData(tags)
			log.Printf("  Debug: Parsed tasks from tags: %v", tasks)
			if len(tasks) > 0 {
				existingMetadata.Tasks = tasks
				enrichmentInfo.DataSources.Tasks = "huggingface.tags"
			}
		}
	}

	// If still no tasks and we have a README, try to infer from model architecture (lowest priority)
	if len(existingMetadata.Tasks) == 0 && existingMetadata.Readme != nil {
		inferredTasks := huggingface.InferTasksFromReadme(*existingMetadata.Readme)
		if len(inferredTasks) > 0 {
			existingMetadata.Tasks = inferredTasks
			enrichmentInfo.DataSources.Tasks = "modelcard.inferred"
		}
	}

	// Handle enriched createTimeSinceEpoch data
	if enrichedData.CreateTimeSinceEpoch.Source != "null" && enrichedData.CreateTimeSinceEpoch.Value != nil {
		if createEpoch, ok := enrichedData.CreateTimeSinceEpoch.Value.(int64); ok {
			// Use enriched createTimeSinceEpoch if not already set or if existing value is null/zero
			if existingMetadata.CreateTimeSinceEpoch == nil || *existingMetadata.CreateTimeSinceEpoch == 0 {
				existingMetadata.CreateTimeSinceEpoch = &createEpoch
				enrichmentInfo.DataSources.CreateTimeSinceEpoch = enrichedData.CreateTimeSinceEpoch.Source
				log.Printf("  Set createTimeSinceEpoch from enriched data: %d", createEpoch)
			}
		}
	}

	// Handle enriched lastUpdateTimeSinceEpoch data from HuggingFace README
	if enrichedData.LastModified.Source != "null" && strings.HasPrefix(enrichedData.LastModified.Source, "huggingface") && enrichedData.LastModified.Value != nil {
		if releaseEpoch, ok := enrichedData.LastModified.Value.(int64); ok {
			// Use README release date for lastUpdateTimeSinceEpoch if not already set or if existing value is null/zero
			if existingMetadata.LastUpdateTimeSinceEpoch == nil || *existingMetadata.LastUpdateTimeSinceEpoch == 0 {
				existingMetadata.LastUpdateTimeSinceEpoch = &releaseEpoch
				enrichmentInfo.DataSources.LastModified = enrichedData.LastModified.Source
				log.Printf("  Set lastUpdateTimeSinceEpoch from HuggingFace README release date: %d", releaseEpoch)
			}
		}
	}

	// Final step: Set license link for any license that doesn't already have one
	if existingMetadata.License != nil && existingMetadata.LicenseLink == nil {
		if licenseURL := utils.GetLicenseURL(*existingMetadata.License); licenseURL != "" {
			existingMetadata.LicenseLink = &licenseURL
		}
	}

	// IMPORTANT: Preserve readme content if it's missing but modelcard file exists
	if existingMetadata.Readme == nil {
		modelcardPath := fmt.Sprintf("output/%s/models/modelcard.md", sanitizedName)
		if modelcardContent, err := os.ReadFile(modelcardPath); err == nil && len(modelcardContent) > 0 {
			readme := string(modelcardContent)
			existingMetadata.Readme = &readme
			log.Printf("  Restored readme content from modelcard.md for: %s", registryModel)
		}
	}

	// DESCRIPTION FALLBACK LOGIC: Generate description if missing
	if existingMetadata.Description == nil {
		var description string

		// First try to get description from HuggingFace data if available
		if enrichedData.HuggingFaceModel != "" {
			// Try to extract description from HuggingFace model name/path
			description = utils.GenerateDescriptionFromModelName(enrichedData.HuggingFaceModel)
		} else if existingMetadata.Name != nil {
			// Fallback to model name if no HuggingFace match
			description = utils.GenerateDescriptionFromModelName(*existingMetadata.Name)
		}

		if description != "" {
			existingMetadata.Description = &description
			log.Printf("  Generated description from model name for: %s", registryModel)
		}
	}

	// Write clean metadata to metadata.yaml (without enrichment section)
	updatedData, err := yaml.Marshal(existingMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal updated metadata: %v", err)
	}

	err = os.WriteFile(metadataPath, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated metadata: %v", err)
	}

	// Write enrichment data to separate enrichment.yaml file
	enrichmentData, err := yaml.Marshal(enrichmentInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal enrichment data: %v", err)
	}

	err = os.WriteFile(enrichmentPath, enrichmentData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write enrichment file: %v", err)
	}

	return nil
}
