// Package memgen generates a trivial in-memory implementation of an
// entity's repository port, used as the default wiring so a generated
// project is runnable end to end before Phase 4 adds real persistence
// adapters (Postgres/MySQL/Mongo) selectable via sgo.yaml.
package memgen

import (
	"embed"
	"path"
	"path/filepath"
	"strings"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// ImportPath is where the in-memory adapter lives, e.g.
// "<module>/internal/infrastructure/persistence/memory" (ARCHITECTURE.md
// §17).
func ImportPath(module string) string {
	return path.Join(module, "internal/infrastructure/persistence/memory")
}

// Generate writes
// internal/infrastructure/persistence/memory/<entity>_repository_gen.go,
// a trivial implementation of the aggregate's repository port
// (core.GenerateAggregateRepositoryPort's `<Entity>Repository` interface
// in internal/domain/<entity> — identical CRUD shape to the port this
// replaced, so this adapter's own Create/Get/List/Update/Delete methods
// don't need to change, only which domain package they operate on).
func Generate(p core.Paths, destDir string) error {
	entity := entityTitle(p.Entity)

	data := struct {
		Entity           string
		EntityLower      string
		DomainPkg        string
		DomainImportPath string
	}{
		Entity:           entity,
		EntityLower:      p.Entity,
		DomainPkg:        p.Entity,
		DomainImportPath: p.AggregateDomainImportPath(),
	}

	destPath := filepath.Join(destDir, p.Entity+"_repository_gen.go")
	return gengo.Write(templatesFS, "templates/memory_repository_gen.go.tmpl", data, destPath)
}

func entityTitle(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
