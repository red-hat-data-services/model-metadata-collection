package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/containers/image/v5/docker"
	blobinfocachememory "github.com/containers/image/v5/pkg/blobinfocache/memory"
	"github.com/containers/image/v5/types"
)

func main() {
	manifestRef := "registry.redhat.io/rhelai1/modelcar-mixtral-8x7b-instruct-v0-1:latest"
	manifestRef = "registry.redhat.io/rhelai1/modelcar-granite-3-1-8b-starter-v2:latest"
	manifestRef = "registry.redhat.io/rhelai1/modelcar-granite-3-1-8b-starter-v2-1:latest"
	manifestRef = "registry.redhat.io/rhelai1/modelcar-granite-3-1-8b-lab-v2-1:latest"
	manifestRef = "registry.redhat.io/rhelai1/modelcar-granite-3-1-8b-base-quantized-w4a16:1.5"

	sys := &types.SystemContext{}

	src, layers := FetchManifestSrcAndLayers(manifestRef, sys)
	defer src.Close()
	ScanLayersForModelCarD(layers, src)
}

func ScanLayersForModelCarD(layers []types.BlobInfo, src types.ImageSource) {
	for i, layer := range layers {
		fmt.Printf("Layer %d:\n", i+1)
		fmt.Printf("  Digest: %s\n", layer.Digest)
		fmt.Printf("  MediaType: %s\n", layer.MediaType)
		fmt.Printf("  Size: %d bytes\n", layer.Size)
		if layer.Annotations != nil {
			fmt.Printf("  Annotations: %v\n", layer.Annotations)

			// Check if this layer has the modelcard annotation
			if layerType, exists := layer.Annotations["io.opendatahub.modelcar.layer.type"]; exists && layerType == "modelcard" {
				fmt.Printf("  Found modelcard layer! Attempting to access modelcard layer blob with digest: %s\n", layer.Digest)

				var layerBlob io.ReadCloser
				var err error

				layerBlob, _, err = src.GetBlob(context.Background(), types.BlobInfo{
					Digest: layer.Digest,
				}, blobinfocachememory.New())
				if err != nil {
					log.Fatalf("Failed to get modelcard layer blob: %v", err)
				}

				if layerBlob == nil {
					log.Printf("layerBlob is nil for modelcard layer")
				} else {
					var reader io.Reader = layerBlob
					defer layerBlob.Close()
					fmt.Printf("  Successfully fetched modelcard layer blob. Attempting to read as tar...\n")

					// Check if it's a gzipped tar file
					if strings.Contains(layer.MediaType, "+gzip") {
						fmt.Printf("  Detected gzipped tar file, decompressing...\n")
						gzReader, err := gzip.NewReader(layerBlob)
						if err != nil {
							log.Printf("Error creating gzip reader: %v", err)
							continue
						}
						defer gzReader.Close()
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
						fmt.Printf("  Found file in tar: %s (size: %d bytes)\n", header.Name, header.Size)
						if strings.HasSuffix(header.Name, ".md") {
							mdFileCount++
							if mdFileCount > 1 {
								fmt.Printf("  Found multiple .md files, skipping content display\n")
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
						fmt.Printf("  Found single .md file: %s (size: %d bytes)\n", singleMdFileName, len(singleMdContent))
						lines := bytes.Split(singleMdContent, []byte("\n"))
						fmt.Printf("  First 10 lines of %s:\n", singleMdFileName)
						fmt.Printf("  ----------------------------------------\n")
						for j, line := range lines {
							if j >= 10 {
								break
							}
							fmt.Printf("  %d: %s\n", j+1, string(line))
						}
						fmt.Printf("  ----------------------------------------\n")
					} else {
						fmt.Printf("  No .md files found in the blob\n")
					}
				}
			}
		}
		fmt.Println()
	}
}

func FetchManifestSrcAndLayers(manifestRef string, sys *types.SystemContext) (types.ImageSource, []types.BlobInfo) {
	fmt.Printf("Parsing reference...\n")
	ref, err := docker.ParseReference("//" + manifestRef)
	if err != nil {
		log.Fatalf("Failed to parse reference: %v", err)
	}

	// Create a new image source (later will use to get "the" blob)
	fmt.Printf("Creating image source...\n")
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

	fmt.Printf("Manifest type: %s\n", manifestType)
	fmt.Printf("Manifest size: %d bytes\n", len(manifest))
	fmt.Println()

	// Get the image
	img, err := ref.NewImage(context.Background(), sys)
	if err != nil {
		log.Fatalf("Failed to create image: %v", err)
	}
	defer img.Close()

	// Get the image configuration
	fmt.Printf("Getting config blob...\n")
	configBlob, err := img.ConfigBlob(context.Background())
	if err != nil {
		log.Fatalf("Failed to get config blob: %v", err)
	}

	fmt.Printf("Config blob size: %d bytes\n", len(configBlob))
	fmt.Println()

	// Get layer information
	fmt.Printf("Getting layer infos...\n")
	layers := img.LayerInfos()
	fmt.Printf("Number of layers: %d\n", len(layers))
	fmt.Println()

	// Get layer digests from layer infos
	fmt.Println("Layer digests:")
	for i, layer := range layers {
		fmt.Printf("  Layer %d: %s\n", i+1, layer.Digest)
	}
	return src, layers
}
