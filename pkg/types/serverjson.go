package types

const ServerJSONSchemaURL = "https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json"

// ServerJSON represents the MCP server.json format (schema version 2025-12-11).
type ServerJSON struct {
	Schema      string                `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	Name        string                `json:"name" yaml:"name"`
	Description string                `json:"description" yaml:"description"`
	Version     string                `json:"version" yaml:"version"`
	Title       string                `json:"title,omitempty" yaml:"title,omitempty"`
	WebsiteURL  string                `json:"websiteUrl,omitempty" yaml:"websiteUrl,omitempty"`
	Repository  *ServerJSONRepository `json:"repository,omitempty" yaml:"repository,omitempty"`
	Packages    []ServerJSONPackage   `json:"packages,omitempty" yaml:"packages,omitempty"`
}

type ServerJSONRepository struct {
	URL    string `json:"url" yaml:"url"`
	Source string `json:"source" yaml:"source"`
	ID     string `json:"id,omitempty" yaml:"id,omitempty"`
}

type ServerJSONPackage struct {
	RegistryType         string                    `json:"registryType" yaml:"registryType"`
	Identifier           string                    `json:"identifier" yaml:"identifier"`
	Transport            ServerJSONTransport       `json:"transport" yaml:"transport"`
	Version              string                    `json:"version,omitempty" yaml:"version,omitempty"`
	RuntimeHint          string                    `json:"runtimeHint,omitempty" yaml:"runtimeHint,omitempty"`
	PackageArguments     []ServerJSONArgument      `json:"packageArguments,omitempty" yaml:"packageArguments,omitempty"`
	EnvironmentVariables []ServerJSONKeyValueInput `json:"environmentVariables,omitempty" yaml:"environmentVariables,omitempty"`
}

type ServerJSONTransport struct {
	Type string `json:"type" yaml:"type"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

type ServerJSONArgument struct {
	Type      string `json:"type" yaml:"type"`
	Value     string `json:"value,omitempty" yaml:"value,omitempty"`
	ValueHint string `json:"valueHint,omitempty" yaml:"valueHint,omitempty"`
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
}

type ServerJSONKeyValueInput struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	IsRequired  *bool  `json:"isRequired,omitempty" yaml:"isRequired,omitempty"`
	IsSecret    *bool  `json:"isSecret,omitempty" yaml:"isSecret,omitempty"`
	Default     string `json:"default,omitempty" yaml:"default,omitempty"`
}
