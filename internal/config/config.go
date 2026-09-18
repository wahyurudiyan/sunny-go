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

// Persistence holds the persistence-related selections for a project.
type Persistence struct {
	Mode    PersistenceMode     `yaml:"mode"`
	Engines []PersistenceEngine `yaml:"engines,omitempty"`
}

// Config is the sgo.yaml project manifest.
type Config struct {
	Module        string         `yaml:"module"`
	HTTPFramework HTTPFramework  `yaml:"httpFramework"`
	Persistence   Persistence    `yaml:"persistence"`
	Cache         []CacheEngine  `yaml:"cache,omitempty"`
	Search        []SearchEngine `yaml:"search,omitempty"`
	Services      []string       `yaml:"services,omitempty"`
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
