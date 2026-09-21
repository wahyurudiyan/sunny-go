// Package cachegen generates the project-scoped (not per-entity) cache
// port and its Redis adapter (ARCHITECTURE.md §8.2/§8.3). Available
// infrastructure once generated, but — unlike the repository port —
// nothing wires it into a service automatically, since that would mean
// changing an owned file's constructor signature (see ARCHITECTURE.md
// §12).
package cachegen

import (
	"embed"
	"path"
	"path/filepath"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// ImportPath is where the Redis adapter lives, e.g.
// "<module>/internal/adapter/out/cache/redis".
func ImportPath(module string) string {
	return path.Join(module, "internal/adapter/out/cache/redis")
}

// GeneratePort writes internal/core/port/out/cache.go.
func GeneratePort(destDir string) error {
	return gengo.Write(templatesFS, "templates/cache_port_gen.go.tmpl", nil, filepath.Join(destDir, "cache.go"))
}

// GenerateRedis writes the Redis adapter into destDir.
func GenerateRedis(destDir string) error {
	if err := gengo.Write(templatesFS, "templates/redis_cache_gen.go.tmpl", nil, filepath.Join(destDir, "cache_gen.go")); err != nil {
		return err
	}
	return gengo.Write(templatesFS, "templates/redis_conn_gen.go.tmpl", nil, filepath.Join(destDir, "conn_gen.go"))
}
