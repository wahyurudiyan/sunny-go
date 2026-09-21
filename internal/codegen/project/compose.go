package project

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// composeService is a single docker-compose service block. Built
// programmatically (not text/template) so the output is always
// well-formed YAML regardless of how many datastores are selected.
type composeService struct {
	Image       string            `yaml:"image"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
}

type composeFile struct {
	Services map[string]composeService `yaml:"services"`
	Volumes  map[string]any            `yaml:"volumes,omitempty"`
}

func writeDockerCompose(destDir string, opts Options) error {
	compose := composeFile{
		Services: map[string]composeService{},
		Volumes:  map[string]any{},
	}

	if hasEngine(opts.Persistence.Engines, config.PersistenceEnginePostgres) {
		compose.Services["postgres"] = composeService{
			Image: "postgres:16-alpine",
			Environment: map[string]string{
				"POSTGRES_USER":     opts.Name,
				"POSTGRES_PASSWORD": opts.Name,
				"POSTGRES_DB":       opts.Name,
			},
			Ports:   []string{"5432:5432"},
			Volumes: []string{"postgres-data:/var/lib/postgresql/data"},
		}
		compose.Volumes["postgres-data"] = nil
	}

	if hasEngine(opts.Persistence.Engines, config.PersistenceEngineMySQL) {
		compose.Services["mysql"] = composeService{
			Image: "mysql:8",
			Environment: map[string]string{
				"MYSQL_ROOT_PASSWORD": opts.Name,
				"MYSQL_DATABASE":      opts.Name,
			},
			Ports:   []string{"3306:3306"},
			Volumes: []string{"mysql-data:/var/lib/mysql"},
		}
		compose.Volumes["mysql-data"] = nil
	}

	if hasEngine(opts.Persistence.Engines, config.PersistenceEngineMongo) {
		compose.Services["mongo"] = composeService{
			Image:   "mongo:7",
			Ports:   []string{"27017:27017"},
			Volumes: []string{"mongo-data:/data/db"},
		}
		compose.Volumes["mongo-data"] = nil
	}

	for _, c := range opts.Cache {
		if c == config.CacheEngineRedis {
			compose.Services["redis"] = composeService{
				Image: "redis:7-alpine",
				Ports: []string{"6379:6379"},
			}
		}
	}

	for _, s := range opts.Search {
		if s == config.SearchEngineElasticsearch {
			compose.Services["elasticsearch"] = composeService{
				Image: "docker.elastic.co/elasticsearch/elasticsearch:8.13.0",
				Environment: map[string]string{
					"discovery.type":         "single-node",
					"xpack.security.enabled": "false",
				},
				Ports: []string{"9200:9200"},
			}
		}
	}

	if len(compose.Volumes) == 0 {
		compose.Volumes = nil
	}

	data, err := yaml.Marshal(compose)
	if err != nil {
		return fmt.Errorf("failed to marshal docker-compose.yml: %w", err)
	}

	path := filepath.Join(destDir, "docker", "docker-compose.yml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

func hasEngine(engines []config.PersistenceEngine, want config.PersistenceEngine) bool {
	for _, e := range engines {
		if e == want {
			return true
		}
	}
	return false
}
