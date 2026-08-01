package catalog

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/opendatahub-io/model-metadata-collection/pkg/types"
)

func buildServerJSON(server *types.MCPServerMetadata) (*types.ServerJSON, error) {
	sj := &types.ServerJSON{
		Schema:      types.ServerJSONSchemaURL,
		Name:        server.Name,
		Description: server.Description,
		Version:     server.Version,
	}

	if server.DisplayName != "" {
		sj.Title = server.DisplayName
	}
	if server.DocumentationUrl != "" {
		sj.WebsiteURL = server.DocumentationUrl
	}

	sj.Repository = parseRepository(server.RepositoryUrl)

	pkgs, err := buildPackages(server)
	if err != nil {
		return nil, err
	}
	sj.Packages = pkgs

	return sj, nil
}

func parseRepository(repoURL string) *types.ServerJSONRepository {
	if repoURL == "" {
		return nil
	}

	repo := &types.ServerJSONRepository{URL: repoURL}

	parsed, err := url.Parse(repoURL)
	if err != nil {
		repo.Source = "git"
		return repo
	}

	host := strings.ToLower(parsed.Host)
	switch {
	case strings.Contains(host, "github.com"):
		repo.Source = "github"
	case strings.Contains(host, "gitlab.com"):
		repo.Source = "gitlab"
	default:
		repo.Source = "git"
	}

	return repo
}

func buildPackages(server *types.MCPServerMetadata) ([]types.ServerJSONPackage, error) {
	if len(server.Artifacts) == 0 {
		return nil, nil
	}

	transport, err := deriveTransport(server)
	if err != nil {
		return nil, err
	}
	args := buildPackageArguments(server.RuntimeMetadata)
	envVars := extractEnvironmentVariables(server.RuntimeMetadata)

	var pkgs []types.ServerJSONPackage
	for _, artifact := range server.Artifacts {
		identifier := strings.TrimPrefix(artifact.URI, "oci://")

		pkg := types.ServerJSONPackage{
			RegistryType: "oci",
			Identifier:   identifier,
			Transport:    transport,
			RuntimeHint:  "docker",
		}
		if len(args) > 0 {
			pkg.PackageArguments = args
		}
		if len(envVars) > 0 {
			pkg.EnvironmentVariables = envVars
		}

		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func deriveTransport(server *types.MCPServerMetadata) (types.ServerJSONTransport, error) {
	port, hasPort := getIntFromMap(server.RuntimeMetadata, "defaultPort")
	if !hasPort {
		return types.ServerJSONTransport{}, fmt.Errorf("server %q: missing required defaultPort in runtimeMetadata", server.Name)
	}

	mcpPath, hasPath := getStringFromMap(server.RuntimeMetadata, "mcpPath")
	if !hasPath {
		mcpPath = "/mcp"
	}

	transportType, err := resolveTransportType(server.Transports)
	if err != nil {
		return types.ServerJSONTransport{}, fmt.Errorf("server %q: %w", server.Name, err)
	}

	return types.ServerJSONTransport{
		Type: transportType,
		URL:  fmt.Sprintf("http://localhost:%d%s", port, mcpPath),
	}, nil
}

// resolveTransportType picks the best transport from the declared list.
// Valid server.json types are: streamable-http, sse.
// Input "http" is normalized to "streamable-http".
// Servers in this catalog are always hosted; stdio-only is invalid data.
func resolveTransportType(transports []string) (string, error) {
	preference := map[string]int{
		"streamable-http": 3,
		"sse":             2,
		"stdio":           1,
	}

	best := ""
	bestRank := 0
	for _, t := range transports {
		normalized := normalizeTransport(t)
		if rank := preference[normalized]; rank > bestRank {
			best = normalized
			bestRank = rank
		}
	}

	if best == "" {
		return "", fmt.Errorf("invalid transport: no transports declared; servers must declare at least one hosted transport (http or sse)")
	}
	if best == "stdio" {
		return "", fmt.Errorf("invalid transport: servers in this catalog must declare a hosted transport (http or sse), got only stdio")
	}
	return best, nil
}

func normalizeTransport(t string) string {
	switch strings.ToLower(t) {
	case "http", "streamable-http":
		return "streamable-http"
	case "sse":
		return "sse"
	case "stdio":
		return "stdio"
	default:
		return strings.ToLower(t)
	}
}

func buildPackageArguments(rm map[string]any) []types.ServerJSONArgument {
	rawArgs, ok := getSliceFromMap(rm, "defaultArgs")
	if !ok || len(rawArgs) == 0 {
		return nil
	}

	var args []types.ServerJSONArgument
	i := 0
	for i < len(rawArgs) {
		s, ok := rawArgs[i].(string)
		if !ok {
			i++
			continue
		}

		if strings.HasPrefix(s, "-") {
			arg := types.ServerJSONArgument{Type: "named", Name: s}
			if i+1 < len(rawArgs) {
				if next, ok := rawArgs[i+1].(string); ok && !strings.HasPrefix(next, "-") {
					arg.Value = next
					i++
				}
			}
			args = append(args, arg)
		} else {
			args = append(args, types.ServerJSONArgument{Type: "positional", Value: s})
		}
		i++
	}
	return args
}

func extractEnvironmentVariables(rm map[string]any) []types.ServerJSONKeyValueInput {
	seen := make(map[string]*types.ServerJSONKeyValueInput)
	var order []string

	addVar := func(name, description, defaultValue string, required, secret bool) {
		if name == "" {
			return
		}
		if existing, ok := seen[name]; ok {
			if secret {
				existing.IsSecret = boolPtr(true)
			}
			if existing.Default == "" {
				existing.Default = defaultValue
			}
			return
		}
		kv := &types.ServerJSONKeyValueInput{
			Name:        name,
			Description: description,
			IsRequired:  boolPtr(required),
			Default:     defaultValue,
		}
		if secret {
			kv.IsSecret = boolPtr(true)
		}
		seen[name] = kv
		order = append(order, name)
	}

	prereqs, _ := getMapFromMap(rm, "prerequisites")

	if envVarsList, ok := getSliceFromMap(prereqs, "environmentVariables"); ok {
		for _, item := range envVarsList {
			ev, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name, _ := ev["name"].(string)
			desc, _ := ev["description"].(string)
			def, _ := ev["default"].(string)
			req := toBool(ev["required"])
			addVar(name, desc, def, req, false)
		}
	}

	if secretsList, ok := getSliceFromMap(prereqs, "secrets"); ok {
		for _, item := range secretsList {
			secret, ok := item.(map[string]any)
			if !ok {
				continue
			}
			keys, ok := getSliceFromMap(secret, "keys")
			if !ok {
				continue
			}
			for _, keyItem := range keys {
				keyMap, ok := keyItem.(map[string]any)
				if !ok {
					continue
				}
				envName, _ := keyMap["envVarName"].(string)
				desc, _ := keyMap["description"].(string)
				def, _ := keyMap["default"].(string)
				req := toBool(keyMap["required"])
				if envName != "" {
					addVar(envName, desc, def, req, true)
				}
			}
		}
	}

	for _, topKey := range []string{"requiredEnvironmentVariables", "optionalEnvironmentVariables"} {
		if varsList, ok := getSliceFromMap(rm, topKey); ok {
			for _, item := range varsList {
				ev, ok := item.(map[string]any)
				if !ok {
					continue
				}
				name, _ := ev["name"].(string)
				desc, _ := ev["description"].(string)
				def, _ := ev["default"].(string)
				req := toBool(ev["required"])
				isSecret := false
				if t, ok := ev["type"].(string); ok && t == "secret" {
					isSecret = true
				}
				addVar(name, desc, def, req, isSecret)
			}
		}
	}

	if len(order) == 0 {
		return nil
	}

	result := make([]types.ServerJSONKeyValueInput, 0, len(order))
	for _, name := range order {
		result = append(result, *seen[name])
	}
	return result
}

// Safe map accessors that handle go-yaml type coercion.

func getStringFromMap(m map[string]any, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func getIntFromMap(m map[string]any, key string) (int, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	}
	return 0, false
}

func getMapFromMap(m map[string]any, key string) (map[string]any, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m[key]
	if !ok {
		return nil, false
	}
	sub, ok := v.(map[string]any)
	return sub, ok
}

func getSliceFromMap(m map[string]any, key string) ([]any, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m[key]
	if !ok {
		return nil, false
	}
	s, ok := v.([]any)
	return s, ok
}

func toBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func boolPtr(b bool) *bool {
	return &b
}
