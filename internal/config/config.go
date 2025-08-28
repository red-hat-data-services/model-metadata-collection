package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/opendatahub-io/model-metadata-collection/pkg/types"
)

// LoadModelsFromYAML reads the models list from the YAML configuration file
func LoadModelsFromYAML(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config types.ModelsConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// Extract URIs from the model entries
	var modelURIs []string
	for _, model := range config.Models {
		modelURIs = append(modelURIs, model.URI)
	}

	return modelURIs, nil
}

// LoadModelsConfigFromYAML reads the full models configuration from the YAML file
func LoadModelsConfigFromYAML(filePath string) ([]types.ModelEntry, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config types.ModelsConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return config.Models, nil
}

// LoadModelsFromVersionIndex loads models from a version-specific index file
func LoadModelsFromVersionIndex(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var versionIndex types.VersionIndex
	err = yaml.Unmarshal(data, &versionIndex)
	if err != nil {
		return nil, err
	}

	var modelRefs []string
	for _, model := range versionIndex.Models {
		// Convert HuggingFace model to container registry format
		// This would need to be mapped to actual container registry URLs
		// For now, we'll use a placeholder format
		modelRef := fmt.Sprintf("registry.redhat.io/rhelai1/modelcar-%s", strings.ToLower(strings.ReplaceAll(model.Name, "/", "-")))
		modelRefs = append(modelRefs, modelRef)
	}

	return modelRefs, nil
}
