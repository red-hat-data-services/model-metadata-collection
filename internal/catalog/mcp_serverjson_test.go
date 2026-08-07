package catalog

import (
	"testing"

	"github.com/opendatahub-io/model-metadata-collection/pkg/types"
)

func TestBuildServerJSON(t *testing.T) {
	server := &types.MCPServerMetadata{
		Name:             "com.example/test-server",
		DisplayName:      "Test Server",
		Provider:         "Test Corp",
		License:          "apache-2.0",
		LicenseLink:      "https://example.com/license",
		Description:      "A test MCP server",
		Version:          "1.2.3",
		DocumentationUrl: "https://docs.example.com",
		RepositoryUrl:    "https://github.com/example/test-server",
		SourceCode:       "example/test-server",
		DeploymentMode:   "local",
		PublishedDate:    "2025-01-01T00:00:00Z",
		Transports:       []string{"http", "sse"},
		Tags:             []string{"test", "example"},
		Tools: []types.MCPTool{
			{Name: "list_items", Description: "List items", AccessType: "read_only"},
		},
		Artifacts: []types.MCPArtifact{
			{URI: "oci://registry.example.com/test-server:1.2"},
		},
		RuntimeMetadata: map[string]any{
			"defaultPort": 8080,
			"mcpPath":     "/mcp",
			"defaultArgs": []any{"--config", "/etc/config.toml"},
			"prerequisites": map[string]any{
				"environmentVariables": []any{
					map[string]any{
						"name":        "API_KEY",
						"description": "API key for the service",
						"required":    true,
					},
				},
				"secrets": []any{
					map[string]any{
						"name": "my-secret",
						"keys": []any{
							map[string]any{
								"key":         "token",
								"envVarName":  "SECRET_TOKEN",
								"description": "Secret authentication token",
								"required":    true,
							},
						},
					},
				},
				"serviceAccount": map[string]any{
					"required":      true,
					"suggestedName": "test-sa",
				},
			},
		},
		SecurityIndicators: map[string]any{
			"readOnlyTools":  true,
			"verifiedSource": true,
		},
		Logo:                     "data:image/svg+xml;base64,abc123",
		CreateTimeSinceEpoch:     "1700000000000",
		LastUpdateTimeSinceEpoch: "1700000001000",
	}

	sj, err := buildServerJSON(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sj.Schema != types.ServerJSONSchemaURL {
		t.Errorf("schema = %q, want %q", sj.Schema, types.ServerJSONSchemaURL)
	}
	if sj.Name != "com.example/test-server" {
		t.Errorf("name = %q, want %q", sj.Name, "com.example/test-server")
	}
	if sj.Description != "A test MCP server" {
		t.Errorf("description = %q", sj.Description)
	}
	if sj.Version != "1.2.3" {
		t.Errorf("version = %q", sj.Version)
	}
	if sj.Title != "Test Server" {
		t.Errorf("title = %q", sj.Title)
	}
	if sj.WebsiteURL != "https://docs.example.com" {
		t.Errorf("websiteUrl = %q", sj.WebsiteURL)
	}

	// Repository
	if sj.Repository == nil {
		t.Fatal("repository is nil")
	}
	if sj.Repository.Source != "github" {
		t.Errorf("repository.source = %q, want %q", sj.Repository.Source, "github")
	}
	if sj.Repository.ID != "" {
		t.Errorf("repository.id should be empty, got %q", sj.Repository.ID)
	}

	// Packages
	if len(sj.Packages) != 1 {
		t.Fatalf("packages length = %d, want 1", len(sj.Packages))
	}
	pkg := sj.Packages[0]
	if pkg.RegistryType != "oci" {
		t.Errorf("package.registryType = %q", pkg.RegistryType)
	}
	if pkg.Identifier != "registry.example.com/test-server:1.2" {
		t.Errorf("package.identifier = %q", pkg.Identifier)
	}
	if pkg.Transport.Type != "streamable-http" {
		t.Errorf("package.transport.type = %q, want streamable-http", pkg.Transport.Type)
	}
	if pkg.Transport.URL != "http://localhost:8080/mcp" {
		t.Errorf("package.transport.url = %q", pkg.Transport.URL)
	}
	if pkg.Version != "" {
		t.Errorf("package.version should be empty for OCI, got %q", pkg.Version)
	}

	// Package arguments
	if len(pkg.PackageArguments) != 1 {
		t.Fatalf("packageArguments length = %d, want 1", len(pkg.PackageArguments))
	}
	if pkg.PackageArguments[0].Type != "named" || pkg.PackageArguments[0].Name != "--config" || pkg.PackageArguments[0].Value != "/etc/config.toml" {
		t.Errorf("packageArguments[0] = %+v", pkg.PackageArguments[0])
	}

	// Environment variables
	if len(pkg.EnvironmentVariables) != 2 {
		t.Fatalf("environmentVariables length = %d, want 2", len(pkg.EnvironmentVariables))
	}
	if pkg.EnvironmentVariables[0].Name != "API_KEY" {
		t.Errorf("envVars[0].name = %q", pkg.EnvironmentVariables[0].Name)
	}
	if pkg.EnvironmentVariables[0].IsSecret != nil {
		t.Errorf("envVars[0] should not be secret")
	}
	if pkg.EnvironmentVariables[1].Name != "SECRET_TOKEN" {
		t.Errorf("envVars[1].name = %q", pkg.EnvironmentVariables[1].Name)
	}
	if pkg.EnvironmentVariables[1].IsSecret == nil || !*pkg.EnvironmentVariables[1].IsSecret {
		t.Error("envVars[1] should be secret")
	}

}

func TestBuildServerJSON_Minimal(t *testing.T) {
	server := &types.MCPServerMetadata{
		Name:        "com.example/minimal",
		Provider:    "Test",
		Description: "Minimal server",
		Version:     "0.1.0",
	}

	sj, err := buildServerJSON(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sj.Name != "com.example/minimal" {
		t.Errorf("name = %q", sj.Name)
	}
	if sj.Title != "" {
		t.Errorf("title should be empty, got %q", sj.Title)
	}
	if sj.Repository != nil {
		t.Error("repository should be nil")
	}
	if len(sj.Packages) != 0 {
		t.Errorf("packages should be empty, got %d", len(sj.Packages))
	}

}

func TestParseRepository(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantNil bool
		source  string
	}{
		{"github url", "https://github.com/openshift/openshift-mcp-server", false, "github"},
		{"gitlab url", "https://gitlab.com/org/project", false, "gitlab"},
		{"other url", "https://bitbucket.org/org/repo", false, "git"},
		{"url with .git suffix", "https://github.com/org/repo.git", false, "github"},
		{"empty url", "", true, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := parseRepository(tc.url)
			if tc.wantNil {
				if repo != nil {
					t.Errorf("expected nil, got %+v", repo)
				}
				return
			}
			if repo == nil {
				t.Fatal("expected non-nil repository")
			}
			if repo.Source != tc.source {
				t.Errorf("source = %q, want %q", repo.Source, tc.source)
			}
		})
	}
}

func TestDeriveTransport(t *testing.T) {
	t.Run("uses declared transport with port and path", func(t *testing.T) {
		server := &types.MCPServerMetadata{
			Transports: []string{"http", "sse"},
			RuntimeMetadata: map[string]any{
				"defaultPort": 8080,
				"mcpPath":     "/mcp",
			},
		}
		tr, err := deriveTransport(server)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr.Type != "streamable-http" {
			t.Errorf("type = %q, want streamable-http (http normalized, outranks sse)", tr.Type)
		}
		if tr.URL != "http://localhost:8080/mcp" {
			t.Errorf("url = %q", tr.URL)
		}
	})

	t.Run("prefers streamable-http over sse", func(t *testing.T) {
		server := &types.MCPServerMetadata{
			Transports: []string{"sse", "streamable-http"},
			RuntimeMetadata: map[string]any{
				"defaultPort": 8080,
				"mcpPath":     "/mcp",
			},
		}
		tr, err := deriveTransport(server)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr.Type != "streamable-http" {
			t.Errorf("type = %q, want streamable-http", tr.Type)
		}
	})

	t.Run("port only, no path defaults to /mcp", func(t *testing.T) {
		server := &types.MCPServerMetadata{
			Transports: []string{"sse"},
			RuntimeMetadata: map[string]any{
				"defaultPort": 9090,
			},
		}
		tr, err := deriveTransport(server)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr.Type != "sse" {
			t.Errorf("type = %q, want sse", tr.Type)
		}
		if tr.URL != "http://localhost:9090/mcp" {
			t.Errorf("url = %q", tr.URL)
		}
	})

	t.Run("no port returns error", func(t *testing.T) {
		server := &types.MCPServerMetadata{
			Name:       "com.example/no-port-server",
			Transports: []string{"http", "sse"},
		}
		_, err := deriveTransport(server)
		if err == nil {
			t.Fatal("expected error for missing defaultPort, got nil")
		}
	})

	t.Run("stdio only returns error", func(t *testing.T) {
		server := &types.MCPServerMetadata{
			Name:       "com.example/stdio-server",
			Transports: []string{"stdio"},
		}
		_, err := deriveTransport(server)
		if err == nil {
			t.Error("expected error for stdio-only transport")
		}
	})

	t.Run("no transports returns error", func(t *testing.T) {
		server := &types.MCPServerMetadata{
			Name: "com.example/no-transports",
		}
		_, err := deriveTransport(server)
		if err == nil {
			t.Error("expected error for missing transports")
		}
	})

	t.Run("no transports with port still returns error", func(t *testing.T) {
		server := &types.MCPServerMetadata{
			Name: "com.example/no-transports-with-port",
			RuntimeMetadata: map[string]any{
				"defaultPort": 9090,
			},
		}
		_, err := deriveTransport(server)
		if err == nil {
			t.Error("expected error for missing transports")
		}
	})

	t.Run("float64 port from JSON unmarshal", func(t *testing.T) {
		server := &types.MCPServerMetadata{
			Transports: []string{"streamable-http"},
			RuntimeMetadata: map[string]any{
				"defaultPort": float64(8000),
				"mcpPath":     "/api",
			},
		}
		tr, err := deriveTransport(server)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr.URL != "http://localhost:8000/api" {
			t.Errorf("url = %q", tr.URL)
		}
	})
}

func TestResolveTransportType(t *testing.T) {
	tests := []struct {
		name       string
		transports []string
		want       string
		wantErr    bool
	}{
		{"empty returns error", nil, "", true},
		{"stdio only returns error", []string{"stdio"}, "", true},
		{"http normalizes to streamable-http", []string{"http"}, "streamable-http", false},
		{"sse only", []string{"sse"}, "sse", false},
		{"streamable-http only", []string{"streamable-http"}, "streamable-http", false},
		{"http and sse prefers streamable-http", []string{"http", "sse"}, "streamable-http", false},
		{"sse and stdio prefers sse", []string{"stdio", "sse"}, "sse", false},
		{"all transports prefers streamable-http", []string{"stdio", "http", "sse", "streamable-http"}, "streamable-http", false},
		{"case insensitive", []string{"SSE", "HTTP"}, "streamable-http", false},
		{"case insensitive sse only", []string{"SSE"}, "sse", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveTransportType(tt.transports)
			if tt.wantErr {
				if err == nil {
					t.Errorf("resolveTransportType(%v) expected error, got %q", tt.transports, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveTransportType(%v) = %q, want %q", tt.transports, got, tt.want)
			}
		})
	}
}

func TestNormalizeTransport(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"http", "streamable-http"},
		{"HTTP", "streamable-http"},
		{"streamable-http", "streamable-http"},
		{"Streamable-HTTP", "streamable-http"},
		{"sse", "sse"},
		{"SSE", "sse"},
		{"stdio", "stdio"},
		{"STDIO", "stdio"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeTransport(tt.input)
			if got != tt.want {
				t.Errorf("normalizeTransport(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildPackageArguments(t *testing.T) {
	t.Run("named args", func(t *testing.T) {
		rm := map[string]any{
			"defaultArgs": []any{"--config", "/etc/config.toml"},
		}
		args := buildPackageArguments(rm)
		if len(args) != 1 {
			t.Fatalf("length = %d, want 1", len(args))
		}
		if args[0].Type != "named" || args[0].Name != "--config" || args[0].Value != "/etc/config.toml" {
			t.Errorf("arg = %+v", args[0])
		}
	})

	t.Run("mixed args", func(t *testing.T) {
		rm := map[string]any{
			"defaultArgs": []any{"./server", "streamable-http", "--port", "8080"},
		}
		args := buildPackageArguments(rm)
		if len(args) != 3 {
			t.Fatalf("length = %d, want 3", len(args))
		}
		if args[0].Type != "positional" || args[0].Value != "./server" {
			t.Errorf("args[0] = %+v", args[0])
		}
		if args[1].Type != "positional" || args[1].Value != "streamable-http" {
			t.Errorf("args[1] = %+v", args[1])
		}
		if args[2].Type != "named" || args[2].Name != "--port" || args[2].Value != "8080" {
			t.Errorf("args[2] = %+v", args[2])
		}
	})

	t.Run("flag without value at end", func(t *testing.T) {
		rm := map[string]any{
			"defaultArgs": []any{"--verbose"},
		}
		args := buildPackageArguments(rm)
		if len(args) != 1 {
			t.Fatalf("length = %d, want 1", len(args))
		}
		if args[0].Type != "named" || args[0].Name != "--verbose" || args[0].Value != "" {
			t.Errorf("arg = %+v", args[0])
		}
	})

	t.Run("nil runtime metadata", func(t *testing.T) {
		args := buildPackageArguments(nil)
		if args != nil {
			t.Errorf("expected nil, got %v", args)
		}
	})

	t.Run("empty args", func(t *testing.T) {
		rm := map[string]any{"defaultArgs": []any{}}
		args := buildPackageArguments(rm)
		if args != nil {
			t.Errorf("expected nil for empty args, got %v", args)
		}
	})
}

func TestExtractEnvironmentVariables(t *testing.T) {
	t.Run("from prerequisites env vars only", func(t *testing.T) {
		rm := map[string]any{
			"prerequisites": map[string]any{
				"environmentVariables": []any{
					map[string]any{"name": "VAR1", "description": "First var", "required": true},
					map[string]any{"name": "VAR2", "description": "Second var", "required": false},
				},
			},
		}
		vars := extractEnvironmentVariables(rm)
		if len(vars) != 2 {
			t.Fatalf("length = %d, want 2", len(vars))
		}
		if vars[0].Name != "VAR1" || !*vars[0].IsRequired {
			t.Errorf("vars[0] = %+v", vars[0])
		}
		if vars[0].IsSecret != nil {
			t.Error("vars[0] should not have isSecret set")
		}
		if vars[1].Name != "VAR2" || *vars[1].IsRequired {
			t.Errorf("vars[1] = %+v", vars[1])
		}
	})

	t.Run("from secrets", func(t *testing.T) {
		rm := map[string]any{
			"prerequisites": map[string]any{
				"secrets": []any{
					map[string]any{
						"name": "creds",
						"keys": []any{
							map[string]any{
								"envVarName":  "DB_PASSWORD",
								"description": "Database password",
								"required":    true,
							},
						},
					},
				},
			},
		}
		vars := extractEnvironmentVariables(rm)
		if len(vars) != 1 {
			t.Fatalf("length = %d, want 1", len(vars))
		}
		if vars[0].Name != "DB_PASSWORD" {
			t.Errorf("name = %q", vars[0].Name)
		}
		if vars[0].IsSecret == nil || !*vars[0].IsSecret {
			t.Error("should be marked as secret")
		}
	})

	t.Run("dedup with secret taking precedence", func(t *testing.T) {
		rm := map[string]any{
			"prerequisites": map[string]any{
				"environmentVariables": []any{
					map[string]any{"name": "TOKEN", "description": "Auth token", "required": true},
				},
				"secrets": []any{
					map[string]any{
						"name": "auth-secret",
						"keys": []any{
							map[string]any{"envVarName": "TOKEN", "description": "Auth token from secret", "required": true},
						},
					},
				},
			},
		}
		vars := extractEnvironmentVariables(rm)
		if len(vars) != 1 {
			t.Fatalf("length = %d, want 1 (deduped)", len(vars))
		}
		if vars[0].IsSecret == nil || !*vars[0].IsSecret {
			t.Error("deduped TOKEN should be marked as secret")
		}
	})

	t.Run("from top-level optional/required env vars", func(t *testing.T) {
		rm := map[string]any{
			"requiredEnvironmentVariables": []any{
				map[string]any{"name": "REQUIRED_VAR", "description": "Required", "required": true},
			},
			"optionalEnvironmentVariables": []any{
				map[string]any{"name": "OPTIONAL_VAR", "description": "Optional", "required": false, "type": "secret"},
			},
		}
		vars := extractEnvironmentVariables(rm)
		if len(vars) != 2 {
			t.Fatalf("length = %d, want 2", len(vars))
		}
		if vars[0].Name != "REQUIRED_VAR" {
			t.Errorf("vars[0].name = %q", vars[0].Name)
		}
		if vars[1].Name != "OPTIONAL_VAR" {
			t.Errorf("vars[1].name = %q", vars[1].Name)
		}
		if vars[1].IsSecret == nil || !*vars[1].IsSecret {
			t.Error("OPTIONAL_VAR with type=secret should be marked as secret")
		}
	})

	t.Run("default values preserved", func(t *testing.T) {
		rm := map[string]any{
			"optionalEnvironmentVariables": []any{
				map[string]any{"name": "DB_PORT", "description": "Database port", "default": "3306"},
				map[string]any{"name": "NO_DEFAULT", "description": "No default value"},
			},
		}
		vars := extractEnvironmentVariables(rm)
		if len(vars) != 2 {
			t.Fatalf("length = %d, want 2", len(vars))
		}
		if vars[0].Default != "3306" {
			t.Errorf("vars[0].Default = %q, want %q", vars[0].Default, "3306")
		}
		if vars[1].Default != "" {
			t.Errorf("vars[1].Default = %q, want empty", vars[1].Default)
		}
	})

	t.Run("dedup fills empty default from later declaration", func(t *testing.T) {
		rm := map[string]any{
			"prerequisites": map[string]any{
				"environmentVariables": []any{
					map[string]any{"name": "HOST", "description": "Host addr"},
				},
			},
			"optionalEnvironmentVariables": []any{
				map[string]any{"name": "HOST", "description": "Host addr", "default": "0.0.0.0"},
			},
		}
		vars := extractEnvironmentVariables(rm)
		if len(vars) != 1 {
			t.Fatalf("length = %d, want 1 (deduped)", len(vars))
		}
		if vars[0].Default != "0.0.0.0" {
			t.Errorf("Default = %q, want %q", vars[0].Default, "0.0.0.0")
		}
	})

	t.Run("nil runtime metadata", func(t *testing.T) {
		vars := extractEnvironmentVariables(nil)
		if vars != nil {
			t.Errorf("expected nil, got %v", vars)
		}
	})
}

func TestMapAccessHelpers(t *testing.T) {
	m := map[string]any{
		"str":    "hello",
		"int":    42,
		"float":  float64(3.14),
		"int64":  int64(100),
		"nested": map[string]any{"key": "val"},
		"list":   []any{"a", "b"},
		"wrong":  true,
	}

	t.Run("getStringFromMap", func(t *testing.T) {
		if v, ok := getStringFromMap(m, "str"); !ok || v != "hello" {
			t.Errorf("got %q, %v", v, ok)
		}
		if _, ok := getStringFromMap(m, "missing"); ok {
			t.Error("expected false for missing key")
		}
		if _, ok := getStringFromMap(m, "int"); ok {
			t.Error("expected false for wrong type")
		}
		if _, ok := getStringFromMap(nil, "str"); ok {
			t.Error("expected false for nil map")
		}
	})

	t.Run("getIntFromMap", func(t *testing.T) {
		if v, ok := getIntFromMap(m, "int"); !ok || v != 42 {
			t.Errorf("got %d, %v", v, ok)
		}
		if v, ok := getIntFromMap(m, "float"); !ok || v != 3 {
			t.Errorf("float64 coercion: got %d, %v", v, ok)
		}
		if v, ok := getIntFromMap(m, "int64"); !ok || v != 100 {
			t.Errorf("int64 coercion: got %d, %v", v, ok)
		}
		if _, ok := getIntFromMap(m, "str"); ok {
			t.Error("expected false for string value")
		}
	})

	t.Run("getMapFromMap", func(t *testing.T) {
		if v, ok := getMapFromMap(m, "nested"); !ok || v["key"] != "val" {
			t.Errorf("got %v, %v", v, ok)
		}
		if _, ok := getMapFromMap(m, "str"); ok {
			t.Error("expected false for wrong type")
		}
	})

	t.Run("getSliceFromMap", func(t *testing.T) {
		if v, ok := getSliceFromMap(m, "list"); !ok || len(v) != 2 {
			t.Errorf("got %v, %v", v, ok)
		}
		if _, ok := getSliceFromMap(m, "str"); ok {
			t.Error("expected false for wrong type")
		}
	})
}
