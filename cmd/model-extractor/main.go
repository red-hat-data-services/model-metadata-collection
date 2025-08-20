package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/containers/image/v5/docker"
	blobinfocachememory "github.com/containers/image/v5/pkg/blobinfocache/memory"
	containertypes "github.com/containers/image/v5/types"
	"gopkg.in/yaml.v3"

	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/catalog"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/config"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/enrichment"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/huggingface"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/metadata"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/internal/registry"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/pkg/types"
	"gitlab.cee.redhat.com/data-hub/model-metadata-collection/pkg/utils"
)

// Command line flags
var (
	modelsIndexPath   = flag.String("input", "data/models-index.yaml", "Path to models index YAML file")
	outputDir         = flag.String("output-dir", "output", "Output directory for extracted metadata")
	catalogOutputPath = flag.String("catalog-output", "data/models-catalog.yaml", "Path for the generated models catalog")
	maxConcurrent     = flag.Int("max-concurrent", 5, "Maximum number of concurrent model processing jobs")
	skipHuggingFace   = flag.Bool("skip-huggingface", false, "Skip HuggingFace collection processing and enrichment")
	skipEnrichment    = flag.Bool("skip-enrichment", false, "Skip metadata enrichment from HuggingFace")
	skipCatalog       = flag.Bool("skip-catalog", false, "Skip catalog generation")
	help              = flag.Bool("help", false, "Show help message")
)

// ModelResult represents the result of processing a single model
type ModelResult struct {
	Ref            string
	ModelCardFound bool
	Metadata       types.ModelMetadata
}

func main() {
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	log.Printf("Starting model metadata collection with configuration:")
	log.Printf("  Models Index: %s", *modelsIndexPath)
	log.Printf("  Output Directory: %s", *outputDir)
	log.Printf("  Catalog Output: %s", *catalogOutputPath)
	log.Printf("  Max Concurrent: %d", *maxConcurrent)
	log.Printf("  Skip HuggingFace: %v", *skipHuggingFace)
	log.Printf("  Skip Enrichment: %v", *skipEnrichment)
	log.Printf("  Skip Catalog: %v", *skipCatalog)

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Ensure catalog output directory exists
	catalogDir := filepath.Dir(*catalogOutputPath)
	if err := os.MkdirAll(catalogDir, 0755); err != nil {
		log.Fatalf("Failed to create catalog output directory: %v", err)
	}

	// Process HuggingFace collections (unless skipped)
	if !*skipHuggingFace {
		log.Println("Processing HuggingFace collections...")
		err := huggingface.ProcessCollections()
		if err != nil {
			log.Printf("Warning: Failed to process HuggingFace collections: %v", err)
			log.Println("Falling back to existing models-index.yaml")
		}
	}

	// Load models from configuration file
	manifestRefs, err := loadModels(*modelsIndexPath)
	if err != nil {
		log.Fatalf("Failed to load models: %v", err)
	}

	log.Printf("Processing %d models...", len(manifestRefs))

	// Process models in parallel
	modelResults := processModelsInParallel(manifestRefs, *maxConcurrent)

	// Generate manifests.yaml
	err = generateManifestsYAML(modelResults, *outputDir)
	if err != nil {
		log.Fatalf("Failed to generate manifests.yaml: %v", err)
	}

	log.Printf("All manifest processing completed")

	// Enrich registry model metadata with HuggingFace data (unless skipped)
	// This happens AFTER model processing to enrich the extracted metadata
	if !*skipEnrichment {
		log.Println("Enriching extracted metadata with HuggingFace data...")
		err := enrichment.EnrichMetadataFromHuggingFace()
		if err != nil {
			log.Printf("Warning: Failed to enrich metadata: %v", err)
		}

		// Update all existing models with OCI artifact metadata
		err = enrichment.UpdateAllModelsWithOCIArtifacts()
		if err != nil {
			log.Printf("Warning: Failed to update OCI artifacts: %v", err)
		}
	}

	// Create the models catalog (unless skipped)
	if !*skipCatalog {
		log.Printf("Creating models catalog...")
		err = catalog.CreateModelsCatalog()
		if err != nil {
			log.Fatalf("Failed to create models catalog: %v", err)
		}
	}

	log.Println("Model metadata collection completed successfully!")
}

func printHelp() {
	fmt.Println("Model Metadata Collection Tool")
	fmt.Println("")
	fmt.Println("This tool extracts metadata from Red Hat AI model containers and enriches it with HuggingFace data.")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Printf("  %s [options]\n", os.Args[0])
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Basic usage with default settings")
	fmt.Printf("  %s\n", os.Args[0])
	fmt.Println("")
	fmt.Println("  # Custom input and output paths")
	fmt.Printf("  %s --input custom-models.yaml --output-dir /tmp/output --catalog-output /tmp/catalog.yaml\n", os.Args[0])
	fmt.Println("")
	fmt.Println("  # Skip HuggingFace processing and enrichment")
	fmt.Printf("  %s --skip-huggingface --skip-enrichment\n", os.Args[0])
	fmt.Println("")
	fmt.Println("  # Process only metadata extraction")
	fmt.Printf("  %s --skip-huggingface --skip-enrichment --skip-catalog\n", os.Args[0])
}

// loadModels loads models from various sources with fallback logic
func loadModels(modelsIndexPath string) ([]string, error) {
	// First try to load from specified models index file
	if _, err := os.Stat(modelsIndexPath); err == nil {
		log.Printf("Loading models from: %s", modelsIndexPath)
		return config.LoadModelsFromYAML(modelsIndexPath)
	}

	// Try to load from latest version index file as fallback
	latestIndexFile, err := getLatestVersionIndexFile()
	if err == nil {
		log.Printf("Using latest version index file: %s", latestIndexFile)
		return config.LoadModelsFromVersionIndex(latestIndexFile)
	}

	return nil, fmt.Errorf("no valid models index file found at %s and no version index files available", modelsIndexPath)
}

// getLatestVersionIndexFile finds the latest version index file
func getLatestVersionIndexFile() (string, error) {
	files, err := filepath.Glob("data/hugging-face-redhat-ai-validated-v*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to find version index files: %v", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no version index files found")
	}

	// Sort files to get the latest version
	// This is a simple sort - for production you might want more sophisticated version comparison
	return files[len(files)-1], nil
}

// processModelsInParallel processes multiple models concurrently
func processModelsInParallel(manifestRefs []string, maxConcurrent int) []ModelResult {
	sys := &containertypes.SystemContext{}

	// Create a WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup

	// Create a semaphore to limit concurrent goroutines
	semaphore := make(chan struct{}, maxConcurrent)

	// Channel to collect results from goroutines
	results := make(chan ModelResult, len(manifestRefs))

	// Process each manifest reference in parallel with concurrency limit
	for _, manifestRef := range manifestRefs {
		// Acquire semaphore (blocks if max goroutines are already running)
		semaphore <- struct{}{}

		wg.Add(1)
		go func(ref string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore when done

			log.Printf("Starting processing for: %s", ref)
			src, layers, configBlob := fetchManifestSrcAndLayers(ref, sys)
			defer func() { _ = src.Close() }()
			modelCardFound, metadata := scanLayersForModelCard(layers, src, ref, configBlob)
			log.Printf("Completed processing for: %s", ref)

			// Send result to channel
			results <- ModelResult{
				Ref:            ref,
				ModelCardFound: modelCardFound,
				Metadata:       metadata,
			}
		}(manifestRef)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)

	// Collect all results
	var modelResults []ModelResult
	for result := range results {
		modelResults = append(modelResults, result)
	}

	return modelResults
}

// scanLayersForModelCard scans container layers for model card content
func scanLayersForModelCard(layers []containertypes.BlobInfo, src containertypes.ImageSource, manifestRef string, configBlob []byte) (bool, types.ModelMetadata) {
	for i, layer := range layers {
		log.Printf("Layer %d:", i+1)
		log.Printf("  Digest: %s", layer.Digest)
		log.Printf("  MediaType: %s", layer.MediaType)
		log.Printf("  Size: %d bytes", layer.Size)
		if layer.Annotations != nil {
			log.Printf("  Annotations: %v", layer.Annotations)

			// Check if this layer has the modelcard annotation
			if layerType, exists := layer.Annotations["io.opendatahub.modelcar.layer.type"]; exists && layerType == "modelcard" {
				log.Printf("  Found modelcard layer! Attempting to access modelcard layer blob with digest: %s", layer.Digest)

				var layerBlob io.ReadCloser
				var err error

				layerBlob, _, err = src.GetBlob(context.Background(), containertypes.BlobInfo{
					Digest: layer.Digest,
				}, blobinfocachememory.New())
				if err != nil {
					log.Fatalf("Failed to get modelcard layer blob: %v", err)
				}

				if layerBlob == nil {
					log.Printf("layerBlob is nil for modelcard layer")
				} else {
					var reader io.Reader = layerBlob
					defer func() { _ = layerBlob.Close() }()
					log.Printf("  Successfully fetched modelcard layer blob. Attempting to read as tar...")

					// Check if it's a gzipped tar file
					if strings.Contains(layer.MediaType, "+gzip") {
						log.Printf("  Detected gzipped tar file, decompressing...")
						gzReader, err := gzip.NewReader(layerBlob)
						if err != nil {
							log.Printf("Error creating gzip reader: %v", err)
							continue
						}
						defer func() { _ = gzReader.Close() }()
						reader = gzReader
					}

					tr := tar.NewReader(reader)
					var mdFileCount int
					var singleMdFileName string
					var singleMdContent []byte

					for {
						header, err := tr.Next()
						if err == io.EOF {
							break
						}
						if err != nil {
							log.Printf("Error reading tar: %v", err)
							break
						}
						log.Printf("  Found file in tar: %s (size: %d bytes)", header.Name, header.Size)
						if strings.HasSuffix(header.Name, ".md") {
							mdFileCount++
							if mdFileCount > 1 {
								log.Printf("  Found multiple .md files, skipping content display")
								break
							}
							singleMdFileName = header.Name
							// Only read content if this is the first (and potentially only) .md file
							var content bytes.Buffer
							_, err := io.Copy(&content, tr)
							if err != nil {
								log.Printf("Error reading %s: %v", header.Name, err)
								continue
							}
							singleMdContent = content.Bytes()
						} else {
							// Skip non-.md files
							_, err := io.Copy(io.Discard, tr)
							if err != nil {
								log.Printf("Error skipping %s: %v", header.Name, err)
								continue
							}
						}
					}

					if mdFileCount == 1 {
						log.Printf("  Found single .md file: %s (size: %d bytes)", singleMdFileName, len(singleMdContent))

						// Create output directory
						sanitizedDir := utils.SanitizeManifestRef(manifestRef)
						outputDir := filepath.Join(*outputDir, sanitizedDir)

						// Create the full directory path for the file (including subdirectories)
						outputFilePath := filepath.Join(outputDir, singleMdFileName)
						outputFileDir := filepath.Dir(outputFilePath)
						err := os.MkdirAll(outputFileDir, 0755)
						if err != nil {
							log.Fatalf("Failed to create output directory: %v", err)
						}

						// Write modelcard content to file
						err = os.WriteFile(outputFilePath, singleMdContent, 0644)
						if err != nil {
							log.Fatalf("Failed to write modelcard content to file: %v", err)
						}

						log.Printf("  Successfully wrote modelcard content to: %s", outputFilePath)

						// Parse metadata from the modelcard content
						metadataFlags := metadata.ParseModelCardMetadata(singleMdContent)

						// Extract actual metadata values
						extractedMetadata := metadata.ExtractMetadataValues(singleMdContent)

						// Populate artifacts with OCI registry metadata and real timestamps
						extractedMetadata.Artifacts = registry.ExtractOCIArtifactsFromRegistry(manifestRef)

						// Extract real timestamps from config blob and update artifacts
						createTime, updateTime := extractTimestampsFromConfig(configBlob)
						for i := range extractedMetadata.Artifacts {
							if extractedMetadata.Artifacts[i].CreateTimeSinceEpoch == nil {
								extractedMetadata.Artifacts[i].CreateTimeSinceEpoch = createTime
							}
							if extractedMetadata.Artifacts[i].LastUpdateTimeSinceEpoch == nil {
								extractedMetadata.Artifacts[i].LastUpdateTimeSinceEpoch = updateTime
							}
						}

						// Generate metadata.yaml file in the same directory
						metadataFilePath := filepath.Join(outputFileDir, "metadata.yaml")
						metadataYaml, err := yaml.Marshal(&extractedMetadata)
						if err != nil {
							log.Printf("Failed to marshal metadata to YAML: %v", err)
						} else {
							err = os.WriteFile(metadataFilePath, metadataYaml, 0644)
							if err != nil {
								log.Printf("Failed to write metadata.yaml: %v", err)
							} else {
								log.Printf("  Successfully wrote metadata.yaml to: %s", metadataFilePath)
							}
						}

						return true, metadataFlags
					} else {
						log.Printf("  No .md files found in the blob")
					}
				}
			}
		}
	}
	return false, types.ModelMetadata{}
}

// fetchManifestSrcAndLayers fetches manifest, layers, and config blob from container registry
func fetchManifestSrcAndLayers(manifestRef string, sys *containertypes.SystemContext) (containertypes.ImageSource, []containertypes.BlobInfo, []byte) {
	log.Printf("Parsing reference...")
	ref, err := docker.ParseReference("//" + manifestRef)
	if err != nil {
		log.Fatalf("Failed to parse reference: %v", err)
	}

	// Create a new image source (later will use to get "the" blob)
	log.Printf("Creating image source...")
	src, err := ref.NewImageSource(context.Background(), sys)
	if err != nil {
		log.Fatalf("Failed to create image source: %v", err)
	}
	// not closing `src` given it is returned to the caller

	// Get the manifest
	manifest, manifestType, err := src.GetManifest(context.Background(), nil)
	if err != nil {
		log.Fatalf("Failed to get manifest: %v", err)
	}

	log.Printf("Manifest type: %s", manifestType)
	log.Printf("Manifest size: %d bytes", len(manifest))

	// Get the image
	img, err := ref.NewImage(context.Background(), sys)
	if err != nil {
		log.Fatalf("Failed to create image: %v", err)
	}
	defer func() { _ = img.Close() }()

	// Get the image configuration
	log.Printf("Getting config blob...")
	configBlob, err := img.ConfigBlob(context.Background())
	if err != nil {
		log.Fatalf("Failed to get config blob: %v", err)
	}

	log.Printf("Config blob size: %d bytes", len(configBlob))

	// Get layer information
	log.Printf("Getting layer infos...")
	layers := img.LayerInfos()
	log.Printf("Number of layers: %d", len(layers))

	// Get layer digests from layer infos
	log.Printf("Layer digests:")
	for i, layer := range layers {
		log.Printf("  Layer %d: %s", i+1, layer.Digest)
	}
	return src, layers, configBlob
}

// OCI Image Config structure for timestamp extraction
type OCIImageConfig struct {
	Created string `json:"created"`
	History []struct {
		Created string `json:"created"`
	} `json:"history"`
}

// extractTimestampsFromConfig extracts creation and update timestamps from OCI config blob
func extractTimestampsFromConfig(configBlob []byte) (*int64, *int64) {
	if len(configBlob) == 0 {
		return nil, nil
	}

	var config OCIImageConfig
	if err := json.Unmarshal(configBlob, &config); err != nil {
		log.Printf("Warning: Failed to parse config blob for timestamps: %v", err)
		return nil, nil
	}

	// Parse creation timestamp
	var createTime *int64
	if config.Created != "" {
		if parsedTime, err := time.Parse(time.RFC3339, config.Created); err == nil {
			epochMs := parsedTime.Unix() * 1000
			createTime = &epochMs
		} else {
			log.Printf("Warning: Failed to parse creation time '%s': %v", config.Created, err)
		}
	}

	// Use the most recent history entry for update time, fallback to creation time
	updateTime := createTime
	if len(config.History) > 0 {
		lastHistoryEntry := config.History[len(config.History)-1]
		if lastHistoryEntry.Created != "" {
			if parsedTime, err := time.Parse(time.RFC3339, lastHistoryEntry.Created); err == nil {
				epochMs := parsedTime.Unix() * 1000
				updateTime = &epochMs
			}
		}
	}

	log.Printf("Extracted timestamps - Create: %v, Update: %v", formatTimestamp(createTime), formatTimestamp(updateTime))
	return createTime, updateTime
}

// formatTimestamp formats a timestamp pointer for logging
func formatTimestamp(ts *int64) string {
	if ts == nil {
		return "nil"
	}
	return time.Unix(*ts/1000, 0).Format(time.RFC3339)
}

// generateManifestsYAML creates a manifests.yaml file tracking all processed models
func generateManifestsYAML(modelResults []ModelResult, outputDir string) error {
	var manifests types.ManifestsData

	for _, result := range modelResults {
		manifest := types.ModelManifest{
			Ref: result.Ref,
			ModelCard: types.ModelCard{
				Present:  result.ModelCardFound,
				Metadata: result.Metadata,
			},
		}
		manifests.Models = append(manifests.Models, manifest)
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&manifests)
	if err != nil {
		return err
	}

	// Ensure output directory exists
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return err
	}

	// Write to file in output directory
	manifestsPath := filepath.Join(outputDir, "manifests.yaml")
	err = os.WriteFile(manifestsPath, yamlData, 0644)
	if err != nil {
		return err
	}

	log.Printf("Generated manifests.yaml with %d models", len(manifests.Models))
	return nil
}
