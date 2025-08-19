package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chambridge/model-metadata-collection/pkg/types"
	"github.com/chambridge/model-metadata-collection/pkg/utils"
)

// RegistryManifest represents container registry manifest metadata
type RegistryManifest struct {
	Config struct {
		Created string `json:"created"`
	} `json:"config"`
	History []struct {
		Created string `json:"created"`
	} `json:"history"`
	Annotations map[string]string `json:"annotations"`
}

// parseRegistryImageRef extracts registry, repository, image name and tag from a registry reference
func parseRegistryImageRef(imageRef string) (registry, repository, imageName, tag string, err error) {
	parts := strings.Split(imageRef, "/")
	if len(parts) < 3 {
		return "", "", "", "", fmt.Errorf("invalid image reference format")
	}

	registry = parts[0]
	repository = parts[1]

	// Handle image name and tag
	imageWithTag := strings.Join(parts[2:], "/")
	if idx := strings.LastIndex(imageWithTag, ":"); idx != -1 {
		imageName = imageWithTag[:idx]
		tag = imageWithTag[idx+1:]
	} else {
		imageName = imageWithTag
		tag = "latest"
	}

	return registry, repository, imageName, tag, nil
}

// FetchRegistryMetadata fetches OCI artifact metadata from registry API
func FetchRegistryMetadata(imageRef string) (*types.OCIArtifact, error) {
	registry, repository, imageName, tag, err := parseRegistryImageRef(imageRef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference: %v", err)
	}

	// Create OCI URI format
	ociURI := fmt.Sprintf("oci://%s/%s/%s:%s", registry, repository, imageName, tag)

	// For Red Hat registry, we can try to fetch manifest metadata
	// This is a simplified implementation - in production you'd need proper authentication
	if strings.Contains(registry, "registry.redhat.io") {
		// Try to fetch manifest via registry API v2
		manifestURL := fmt.Sprintf("https://%s/v2/%s/%s/manifests/%s", registry, repository, imageName, tag)

		resp, err := http.Get(manifestURL)
		if err != nil {
			// If we can't fetch from API, create artifact with current timestamp
			currentTime := time.Now().Unix() * 1000
			return &types.OCIArtifact{
				URI:                      ociURI,
				CreateTimeSinceEpoch:     &currentTime,
				LastUpdateTimeSinceEpoch: &currentTime,
				CustomProperties: map[string]interface{}{
					"source": map[string]interface{}{
						"string_value": "registry.redhat.io",
					},
					"type": map[string]interface{}{
						"string_value": "modelcar",
					},
				},
			}, nil
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == 200 {
			// Parse manifest to extract timestamps
			body, err := io.ReadAll(resp.Body)
			if err == nil {
				var manifest RegistryManifest
				if json.Unmarshal(body, &manifest) == nil {
					// Convert timestamps to epoch milliseconds
					createTime := utils.ParseTimeToEpochInt64(manifest.Config.Created)

					// Use the most recent history entry for update time
					updateTime := createTime
					if len(manifest.History) > 0 {
						lastHistoryTime := utils.ParseTimeToEpochInt64(manifest.History[len(manifest.History)-1].Created)
						if lastHistoryTime != nil {
							updateTime = lastHistoryTime
						}
					}

					customProps := map[string]interface{}{
						"source": map[string]interface{}{
							"string_value": "registry.redhat.io",
						},
						"type": map[string]interface{}{
							"string_value": "modelcar",
						},
					}

					// Add annotations as custom properties
					for key, value := range manifest.Annotations {
						customProps[key] = map[string]interface{}{
							"string_value": value,
						}
					}

					return &types.OCIArtifact{
						URI:                      ociURI,
						CreateTimeSinceEpoch:     createTime,
						LastUpdateTimeSinceEpoch: updateTime,
						CustomProperties:         customProps,
					}, nil
				}
			}
		}
	}

	// Fallback: create artifact with current timestamp
	currentTime := time.Now().Unix() * 1000
	return &types.OCIArtifact{
		URI:                      ociURI,
		CreateTimeSinceEpoch:     &currentTime,
		LastUpdateTimeSinceEpoch: &currentTime,
		CustomProperties: map[string]interface{}{
			"source": map[string]interface{}{
				"string_value": registry,
			},
			"type": map[string]interface{}{
				"string_value": "modelcar",
			},
		},
	}, nil
}

// ExtractOCIArtifactsFromRegistry creates structured OCI artifacts from registry references
func ExtractOCIArtifactsFromRegistry(manifestRef string) []types.OCIArtifact {
	var artifacts []types.OCIArtifact

	// The manifestRef itself is the primary OCI artifact
	if artifact, err := FetchRegistryMetadata(manifestRef); err == nil {
		artifacts = append(artifacts, *artifact)
	} else {
		log.Printf("Warning: Failed to fetch registry metadata for %s: %v", manifestRef, err)
		// Create basic artifact anyway
		registry, repository, imageName, tag, parseErr := parseRegistryImageRef(manifestRef)
		if parseErr == nil {
			ociURI := fmt.Sprintf("oci://%s/%s/%s:%s", registry, repository, imageName, tag)
			currentTime := time.Now().Unix() * 1000
			artifacts = append(artifacts, types.OCIArtifact{
				URI:                      ociURI,
				CreateTimeSinceEpoch:     &currentTime,
				LastUpdateTimeSinceEpoch: &currentTime,
				CustomProperties: map[string]interface{}{
					"source": map[string]interface{}{
						"string_value": "unknown",
					},
					"error": map[string]interface{}{
						"string_value": err.Error(),
					},
				},
			})
		}
	}

	// Ensure we never return nil - always return at least an empty slice
	if artifacts == nil {
		artifacts = []types.OCIArtifact{}
	}
	return artifacts
}
