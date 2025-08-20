package enrichment

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"

	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/config"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/huggingface"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/metadata"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/registry"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/pkg/types"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/pkg/utils"
)

// EnrichMetadataFromHuggingFace enriches registry model metadata using HuggingFace data
func EnrichMetadataFromHuggingFace() error {
	log.Println("Enriching registry model metadata with HuggingFace data...")

	// Load HuggingFace models
	hfFilePath := "data/hugging-face-redhat-ai-validated-v1-0.yaml"
	hfData, err := os.ReadFile(hfFilePath)
	if err != nil {
		return fmt.Errorf("failed to read HuggingFace index: %v", err)
	}

	var hfIndex types.VersionIndex
	err = yaml.Unmarshal(hfData, &hfIndex)
	if err != nil {
		return fmt.Errorf("failed to parse HuggingFace index: %v", err)
	}

	// Load registry models
	regModels, err := config.LoadModelsFromYAML("data/models-index.yaml")
	if err != nil {
		return fmt.Errorf("failed to load registry models: %v", err)
	}

	matchCount := 0

	// For each registry model, find the best HuggingFace match and enrich metadata
	for _, regModel := range regModels {
		log.Printf("Processing model: %s", regModel)

		enriched := types.EnrichedModelMetadata{
			RegistryModel:    regModel,
			EnrichmentStatus: "no_match",
		}

		// Try to load existing modelcard metadata
		existingMetadata, err := metadata.LoadExistingMetadata(regModel)
		if err != nil {
			log.Printf("  No existing metadata found for %s", regModel)
		}

		// Initialize metadata sources with existing data or nulls
		enriched.Name = metadata.CreateMetadataSource(nil, "null")
		enriched.Provider = metadata.CreateMetadataSource(nil, "null")
		enriched.Description = metadata.CreateMetadataSource(nil, "null")
		enriched.License = metadata.CreateMetadataSource(nil, "null")
		enriched.LibraryName = metadata.CreateMetadataSource(nil, "null")
		enriched.LastModified = metadata.CreateMetadataSource(nil, "null")
		enriched.CreateTimeSinceEpoch = metadata.CreateMetadataSource(nil, "null")
		enriched.Tags = metadata.CreateMetadataSource(nil, "null")
		enriched.Tasks = metadata.CreateMetadataSource(nil, "null")
		enriched.Downloads = metadata.CreateMetadataSource(nil, "null")
		enriched.Likes = metadata.CreateMetadataSource(nil, "null")
		enriched.ModelSize = metadata.CreateMetadataSource(nil, "null")

		// Populate from existing modelcard metadata if available (only for non-empty values)
		// We need to determine if the data came from YAML frontmatter or text parsing
		if existingMetadata != nil {
			// Try to load the modelcard.md file to analyze the source
			sanitizedName := utils.SanitizeManifestRef(regModel)
			modelcardPath := fmt.Sprintf("output/%s/models/modelcard.md", sanitizedName)

			var modelcardContent string
			var hasYAMLFrontmatter bool
			if content, err := os.ReadFile(modelcardPath); err == nil {
				modelcardContent = string(content)
				// Check if modelcard has YAML frontmatter
				if frontmatter, err := metadata.ExtractYAMLFrontmatterFromModelCard(modelcardContent); err == nil {
					hasYAMLFrontmatter = true
					// Determine sources based on YAML frontmatter presence
					// Note: Only check fields that exist in ModelCardYAMLFrontmatter struct

					// Name and Provider are not in YAML frontmatter, so they're always from regex/text
					if existingMetadata.Name != nil && *existingMetadata.Name != "" {
						enriched.Name = metadata.CreateMetadataSource(*existingMetadata.Name, "modelcard.regex")
					}
					if existingMetadata.Provider != nil && *existingMetadata.Provider != "" {
						enriched.Provider = metadata.CreateMetadataSource(*existingMetadata.Provider, "modelcard.regex")
					}
					if existingMetadata.Description != nil && *existingMetadata.Description != "" {
						enriched.Description = metadata.CreateMetadataSource(*existingMetadata.Description, "modelcard.regex")
					}

					// License can come from YAML frontmatter
					if existingMetadata.License != nil && *existingMetadata.License != "" {
						source := "modelcard.regex"
						if frontmatter.License != "" && frontmatter.License == *existingMetadata.License {
							source = "modelcard.yaml"
						} else if frontmatter.LicenseName != "" && frontmatter.LicenseName == *existingMetadata.License {
							source = "modelcard.yaml"
						}
						enriched.License = metadata.CreateMetadataSource(*existingMetadata.License, source)
					}

					// LibraryName can come from YAML frontmatter
					if existingMetadata.LibraryName != nil && *existingMetadata.LibraryName != "" {
						source := "modelcard.regex"
						if frontmatter.LibraryName != "" && frontmatter.LibraryName == *existingMetadata.LibraryName {
							source = "modelcard.yaml"
						}
						enriched.LibraryName = metadata.CreateMetadataSource(*existingMetadata.LibraryName, source)
					}

					// Tasks can come from YAML frontmatter (pipeline_tag)
					if len(existingMetadata.Tasks) > 0 {
						source := "modelcard.regex"
						if frontmatter.PipelineTag != "" && len(existingMetadata.Tasks) == 1 && existingMetadata.Tasks[0] == frontmatter.PipelineTag {
							source = "modelcard.yaml"
						}
						enriched.Tasks = metadata.CreateMetadataSource(existingMetadata.Tasks, source)
					}
				}
			}

			// If no YAML frontmatter analysis was possible, assume all modelcard data comes from regex/text parsing
			if !hasYAMLFrontmatter {
				if existingMetadata.Name != nil && *existingMetadata.Name != "" {
					enriched.Name = metadata.CreateMetadataSource(*existingMetadata.Name, "modelcard.regex")
				}
				if existingMetadata.Provider != nil && *existingMetadata.Provider != "" {
					enriched.Provider = metadata.CreateMetadataSource(*existingMetadata.Provider, "modelcard.regex")
				}
				if existingMetadata.Description != nil && *existingMetadata.Description != "" {
					enriched.Description = metadata.CreateMetadataSource(*existingMetadata.Description, "modelcard.regex")
				}
				if existingMetadata.License != nil && *existingMetadata.License != "" {
					enriched.License = metadata.CreateMetadataSource(*existingMetadata.License, "modelcard.regex")
				}
				if existingMetadata.LibraryName != nil && *existingMetadata.LibraryName != "" {
					enriched.LibraryName = metadata.CreateMetadataSource(*existingMetadata.LibraryName, "modelcard.regex")
				}
				if len(existingMetadata.Tasks) > 0 {
					enriched.Tasks = metadata.CreateMetadataSource(existingMetadata.Tasks, "modelcard.regex")
				}
			}

			// Handle timestamps (these are typically from text parsing, not YAML)
			if existingMetadata.LastUpdateTimeSinceEpoch != nil {
				enriched.LastModified = metadata.CreateMetadataSource(*existingMetadata.LastUpdateTimeSinceEpoch, "modelcard.regex")
			}
			if existingMetadata.CreateTimeSinceEpoch != nil {
				enriched.CreateTimeSinceEpoch = metadata.CreateMetadataSource(*existingMetadata.CreateTimeSinceEpoch, "modelcard.regex")
			}
		}

		// Find best matching HuggingFace model
		bestMatch := types.ModelIndex{}
		bestScore := 0.0

		for _, hfModel := range hfIndex.Models {
			score := utils.CalculateSimilarity(regModel, hfModel.Name)
			if score > bestScore {
				bestScore = score
				bestMatch = hfModel
			}
		}

		// Enrich with HuggingFace data if we found a good match
		threshold := 0.5
		if bestScore >= threshold {
			enriched.HuggingFaceModel = bestMatch.Name
			enriched.HuggingFaceURL = bestMatch.URL
			enriched.ReadmePath = bestMatch.ReadmePath
			enriched.EnrichmentStatus = "enriched"

			// Set confidence level
			if bestScore >= 0.8 {
				enriched.MatchConfidence = "high"
			} else {
				enriched.MatchConfidence = "medium"
			}

			// Try to fetch detailed HuggingFace metadata
			log.Printf("  Fetching HuggingFace details for: %s", bestMatch.Name)
			hfDetails, err := huggingface.FetchModelDetails(bestMatch.Name)
			if err != nil {
				log.Printf("  Warning: Failed to fetch HF details: %v", err)
			} else {
				// Enrich with HuggingFace data (only if not already available from modelcard)
				if enriched.Name.Source == "null" && hfDetails.ID != "" {
					enriched.Name = metadata.CreateMetadataSource(hfDetails.ID, "huggingface.api")
				}
				if enriched.License.Source == "null" && hfDetails.License != "" {
					enriched.License = metadata.CreateMetadataSource(hfDetails.License, "huggingface.api")
				}
				if enriched.LastModified.Source == "null" && hfDetails.LastModified != "" {
					enriched.LastModified = metadata.CreateMetadataSource(hfDetails.LastModified, "huggingface.api")
				}
				if enriched.Tags.Source == "null" && len(hfDetails.Tags) > 0 {
					enriched.Tags = metadata.CreateMetadataSource(hfDetails.Tags, "huggingface.tags")

					// Parse tags for structured data and potentially extract license
					languages, tagLicense, tasks := huggingface.ParseTagsForStructuredData(hfDetails.Tags)
					log.Printf("  Parsed from tags - Languages: %v, License: %s, Tasks: %v", languages, tagLicense, tasks)

					// Use license from tags if not already set
					if enriched.License.Source == "null" && tagLicense != "" {
						enriched.License = metadata.CreateMetadataSource(tagLicense, "huggingface.tags")
					}

					// Store tasks if found
					if enriched.Tasks.Source == "null" && len(tasks) > 0 {
						enriched.Tasks = metadata.CreateMetadataSource(tasks, "huggingface.tags")
					}
				}
				if enriched.Downloads.Source == "null" && hfDetails.Downloads > 0 {
					enriched.Downloads = metadata.CreateMetadataSource(hfDetails.Downloads, "huggingface.api")
				}
				if enriched.Likes.Source == "null" && hfDetails.Likes > 0 {
					enriched.Likes = metadata.CreateMetadataSource(hfDetails.Likes, "huggingface.api")
				}
			}

			// Try to fetch HuggingFace README for provider, date, and YAML frontmatter information
			needsProvider := enriched.Provider.Source == "null"
			// Extract release date if we don't have a valid date yet (even from modelcard.regex with null value)
			needsReleaseDate := enriched.LastModified.Source == "null" ||
				(enriched.LastModified.Source == "modelcard.regex" && enriched.LastModified.Value == nil)
			needsLibraryName := true // Always try to get library name from YAML
			needsLanguageFromYAML := len(enriched.Tags.Value.([]string)) == 0 || enriched.Tags.Source == "null"

			log.Printf("  DEBUG: LastModified source='%s', value=%v, needsReleaseDate=%v",
				enriched.LastModified.Source, enriched.LastModified.Value, needsReleaseDate)

			if needsProvider || needsReleaseDate || needsLibraryName || needsLanguageFromYAML {
				log.Printf("  Fetching HuggingFace README for additional metadata: %s", bestMatch.Name)
				hfReadme, err := huggingface.FetchReadme(bestMatch.Name)
				if err != nil {
					log.Printf("  Warning: Failed to fetch HF README: %v", err)
				} else {
					// Try to extract YAML frontmatter first
					frontmatter, err := huggingface.ExtractYAMLFrontmatter(hfReadme)
					if err == nil {
						log.Printf("  Successfully extracted YAML frontmatter from HF README")

						// Use library name from YAML
						if frontmatter.LibraryName != "" {
							enriched.LibraryName = metadata.CreateMetadataSource(frontmatter.LibraryName, "huggingface.yaml")
							log.Printf("  Found library_name in YAML frontmatter: %s", frontmatter.LibraryName)
						}

						// Use language from YAML frontmatter if more comprehensive
						if len(frontmatter.Language) > 0 {
							log.Printf("  Found languages in YAML frontmatter: %v", frontmatter.Language)
							// This will override tag-based language detection with more reliable YAML data
						}

						// Use license from YAML frontmatter
						if frontmatter.License != "" && enriched.License.Source == "null" {
							enriched.License = metadata.CreateMetadataSource(frontmatter.License, "huggingface.yaml")
							log.Printf("  Extracted license from YAML frontmatter: %s", frontmatter.License)
						}

						// Use license_name if available and more specific
						if frontmatter.LicenseName != "" {
							enriched.License = metadata.CreateMetadataSource(frontmatter.LicenseName, "huggingface.yaml")
							log.Printf("  Extracted license_name from YAML frontmatter: %s", frontmatter.LicenseName)
						}

						// Use pipeline_tag for tasks
						if frontmatter.PipelineTag != "" && enriched.Tasks.Source == "null" {
							tasks := []string{frontmatter.PipelineTag}
							enriched.Tasks = metadata.CreateMetadataSource(tasks, "huggingface.yaml")
							log.Printf("  Extracted pipeline_tag from YAML frontmatter: %s", frontmatter.PipelineTag)
						}
					} else {
						log.Printf("  No valid YAML frontmatter found in HF README: %v", err)
					}

					// Fallback to text parsing for provider if needed
					if needsProvider && enriched.Provider.Source == "null" {
						provider := huggingface.ExtractProviderFromReadme(hfReadme)
						if provider != "" {
							enriched.Provider = metadata.CreateMetadataSource(provider, "huggingface.regex")
							log.Printf("  Extracted provider from HF README text: %s", provider)
						}
					}

					// Try to extract explicit release date from README (high priority)
					releaseDate := huggingface.ExtractReleaseDateFromReadme(hfReadme)
					if releaseDate != "" {
						if epoch := utils.ParseDateToEpoch(releaseDate); epoch != nil {
							// Use this for createTimeSinceEpoch if we don't have it from modelcard
							if enriched.CreateTimeSinceEpoch.Source == "null" {
								enriched.CreateTimeSinceEpoch = metadata.CreateMetadataSource(*epoch, "huggingface.regex")
								log.Printf("  Extracted createTimeSinceEpoch from HF README release date: %s (epoch: %d)", releaseDate, *epoch)
							}
							// Also update lastModified if we don't have a more recent one
							if needsReleaseDate {
								enriched.LastModified = metadata.CreateMetadataSource(*epoch, "huggingface.regex")
								log.Printf("  Extracted lastModified from HF README release date: %s (epoch: %d)", releaseDate, *epoch)
							}
						}
					}
				}
			}

			// Update the model's metadata.yaml file with enriched data
			err = UpdateModelMetadataFile(regModel, &enriched)
			if err != nil {
				log.Printf("  Warning: Failed to update metadata file for %s: %v", regModel, err)
			} else {
				log.Printf("  Successfully updated metadata file for: %s", regModel)

				// Also update artifacts with OCI metadata
				log.Printf("  Updating OCI artifacts for: %s", regModel)
				err = UpdateOCIArtifacts(regModel)
				if err != nil {
					log.Printf("  Warning: Failed to update OCI artifacts for %s: %v", regModel, err)
				} else {
					log.Printf("  Successfully updated OCI artifacts for: %s", regModel)
				}
			}

			matchCount++
		}

	}

	// Clean up the old enriched metadata file if it exists
	_ = os.Remove("data/enriched-model-metadata.yaml")

	enrichmentRate := float64(matchCount) / float64(len(regModels)) * 100

	log.Printf("Metadata enrichment complete:")
	log.Printf("- Total registry models: %d", len(regModels))
	log.Printf("- Successfully enriched: %d (%.1f%%)", matchCount, enrichmentRate)
	log.Printf("- Individual metadata.yaml files have been updated with enriched data")

	return nil
}

// UpdateAllModelsWithOCIArtifacts updates all existing models with OCI artifact metadata
func UpdateAllModelsWithOCIArtifacts() error {
	log.Println("Updating all existing models with OCI artifact metadata...")

	// Load all models from the index
	regModels, err := config.LoadModelsFromYAML("data/models-index.yaml")
	if err != nil {
		return fmt.Errorf("failed to load registry models: %v", err)
	}

	updateCount := 0

	// Update each model that has existing metadata
	for _, regModel := range regModels {
		// Check if metadata file exists
		sanitizedName := utils.SanitizeManifestRef(regModel)
		metadataPath := fmt.Sprintf("output/%s/models/metadata.yaml", sanitizedName)

		if _, err := os.Stat(metadataPath); err == nil {
			log.Printf("  Updating OCI artifacts for: %s", regModel)
			err = UpdateOCIArtifacts(regModel)
			if err != nil {
				log.Printf("  Warning: Failed to update OCI artifacts for %s: %v", regModel, err)
			} else {
				log.Printf("  Successfully updated OCI artifacts for: %s", regModel)
				updateCount++
			}
		} else {
			log.Printf("  No existing metadata found for: %s", regModel)
		}
	}

	log.Printf("OCI artifacts update complete:")
	log.Printf("- Total models checked: %d", len(regModels))
	log.Printf("- Successfully updated: %d", updateCount)

	return nil
}

// UpdateOCIArtifacts updates the artifacts field with proper OCI metadata for existing models
func UpdateOCIArtifacts(registryModel string) error {
	// Load existing metadata
	existingMetadata, err := metadata.LoadExistingMetadata(registryModel)
	if err != nil {
		return fmt.Errorf("failed to load existing metadata: %v", err)
	}

	// Generate OCI artifacts from the registry model reference
	ociArtifacts := registry.ExtractOCIArtifactsFromRegistry(registryModel)
	existingMetadata.Artifacts = ociArtifacts

	// Write updated metadata back to file
	sanitizedName := utils.SanitizeManifestRef(registryModel)
	metadataPath := fmt.Sprintf("output/%s/models/metadata.yaml", sanitizedName)

	updatedData, err := yaml.Marshal(existingMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal updated metadata: %v", err)
	}

	err = os.WriteFile(metadataPath, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated metadata: %v", err)
	}

	return nil
}
