// Package config reads and writes the sgo.yaml project manifest described
// in ARCHITECTURE.md §7 — the standing record of a project's HTTP
// framework, persistence mode, and datastore choices, so later `sgo
// generate` runs don't need to re-prompt for them.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileName is the manifest file name written to a project's root.
const FileName = "sgo.yaml"

// HTTPFramework identifies the HTTP adapter a project generates routes for.
type HTTPFramework string

const (
	HTTPFrameworkGin  HTTPFramework = "gin"
	HTTPFrameworkEcho HTTPFramework = "echo"
	HTTPFrameworkChi  HTTPFramework = "chi"
)

// PersistenceMode selects whether generated repositories are hand-written
// (self-managed) or backed by an ORM.
type PersistenceMode string

const (
	PersistenceModeORM         PersistenceMode = "orm"
	PersistenceModeSelfManaged PersistenceMode = "self-managed"
)

// PersistenceEngine identifies a datastore behind the repository port.
type PersistenceEngine string

const (
	PersistenceEnginePostgres PersistenceEngine = "postgres"
	PersistenceEngineMySQL    PersistenceEngine = "mysql"
	PersistenceEngineMongo    PersistenceEngine = "mongo"
)

// CacheEngine identifies a datastore behind the cache port.
type CacheEngine string

const CacheEngineRedis CacheEngine = "redis"

// SearchEngine identifies a datastore behind the search port.
type SearchEngine string

const SearchEngineElasticsearch SearchEngine = "elasticsearch"

// OpenAPIVersion selects which OpenAPI Specification version `sgo
// generate openapi` targets (ARCHITECTURE.md §13). The two are close
// enough structurally that openapigen shares one document model between
// them; this only picks which meta-schema validates the result and
// which version string is stamped on it.
type OpenAPIVersion string

const (
	OpenAPIVersion30 OpenAPIVersion = "3.0"
	OpenAPIVersion31 OpenAPIVersion = "3.1"
)

// OpenAPIFormat selects the serialization `sgo generate openapi` writes
// docs/openapi.<ext> in.
type OpenAPIFormat string

const (
	OpenAPIFormatYAML OpenAPIFormat = "yaml"
	OpenAPIFormatJSON OpenAPIFormat = "json"
)

// Persistence holds the persistence-related selections for a project.
type Persistence struct {
	Mode    PersistenceMode     `yaml:"mode" json:"mode"`
	Engines []PersistenceEngine `yaml:"engines,omitempty" json:"engines,omitempty"`
}

// OpenAPI holds a project's OpenAPI documentation generation settings.
type OpenAPI struct {
	Version OpenAPIVersion `yaml:"version" json:"version"`
	Format  OpenAPIFormat  `yaml:"format" json:"format"`
}

// Config is the sgo.yaml project manifest. JSON tags are for the web
// UI's REST API (internal/webui), which serves this struct directly
// rather than keeping a separate response shape in sync with it.
type Config struct {
	Module        string         `yaml:"module" json:"module"`
	HTTPFramework HTTPFramework  `yaml:"httpFramework" json:"httpFramework"`
	Persistence   Persistence    `yaml:"persistence" json:"persistence"`
	Cache         []CacheEngine  `yaml:"cache,omitempty" json:"cache,omitempty"`
	Search        []SearchEngine `yaml:"search,omitempty" json:"search,omitempty"`
	OpenAPI       OpenAPI        `yaml:"openapi" json:"openapi"`
	Services      []string       `yaml:"services,omitempty" json:"services,omitempty"`
}

// DefaultOpenAPI is what `sgo init` persists when the user doesn't pass
// --openapi-version/--openapi-format: 3.0 is the more widely supported
// target for existing tooling (Swagger UI, Redoc, most API gateways),
// yaml matches the rest of sgo's own generated/hand-authored files.
func DefaultOpenAPI() OpenAPI {
	return OpenAPI{Version: OpenAPIVersion30, Format: OpenAPIFormatYAML}
}

var (
	validHTTPFrameworks = map[HTTPFramework]bool{
		HTTPFrameworkGin:  true,
		HTTPFrameworkEcho: true,
		HTTPFrameworkChi:  true,
	}
	validPersistenceModes = map[PersistenceMode]bool{
		PersistenceModeORM:         true,
		PersistenceModeSelfManaged: true,
	}
	validPersistenceEngines = map[PersistenceEngine]bool{
		PersistenceEnginePostgres: true,
		PersistenceEngineMySQL:    true,
		PersistenceEngineMongo:    true,
	}
	validCacheEngines = map[CacheEngine]bool{
		CacheEngineRedis: true,
	}
	validSearchEngines = map[SearchEngine]bool{
		SearchEngineElasticsearch: true,
	}
	validOpenAPIVersions = map[OpenAPIVersion]bool{
		OpenAPIVersion30: true,
		OpenAPIVersion31: true,
	}
	validOpenAPIFormats = map[OpenAPIFormat]bool{
		OpenAPIFormatYAML: true,
		OpenAPIFormatJSON: true,
	}
)

// Validate checks that Config holds only recognized values and required
// fields are set. It does not check filesystem state.
func (c *Config) Validate() error {
	if c.Module == "" {
		return fmt.Errorf("module is required")
	}

	if !validHTTPFrameworks[c.HTTPFramework] {
		return fmt.Errorf("invalid httpFramework %q: must be one of gin, echo, chi", c.HTTPFramework)
	}

	if !validPersistenceModes[c.Persistence.Mode] {
		return fmt.Errorf("invalid persistence.mode %q: must be one of orm, self-managed", c.Persistence.Mode)
	}

	for _, engine := range c.Persistence.Engines {
		if !validPersistenceEngines[engine] {
			return fmt.Errorf("invalid persistence engine %q: must be one of postgres, mysql, mongo", engine)
		}
	}

	for _, engine := range c.Cache {
		if !validCacheEngines[engine] {
			return fmt.Errorf("invalid cache engine %q: must be one of redis", engine)
		}
	}

	for _, engine := range c.Search {
		if !validSearchEngines[engine] {
			return fmt.Errorf("invalid search engine %q: must be one of elasticsearch", engine)
		}
	}

	if !validOpenAPIVersions[c.OpenAPI.Version] {
		return fmt.Errorf("invalid openapi.version %q: must be one of 3.0, 3.1", c.OpenAPI.Version)
	}

	if !validOpenAPIFormats[c.OpenAPI.Format] {
		return fmt.Errorf("invalid openapi.format %q: must be one of yaml, json", c.OpenAPI.Format)
	}

	return nil
}

// Save writes c to <dir>/sgo.yaml, overwriting any existing file.
func (c *Config) Save(dir string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

// Load reads and parses <dir>/sgo.yaml.
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, FileName)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return &cfg, nil
}
