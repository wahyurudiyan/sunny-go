// Package memgen generates a trivial in-memory implementation of an
// entity's repository port, used as the default wiring so a generated
// project is runnable end to end before Phase 4 adds real persistence
// adapters (Postgres/MySQL/Mongo) selectable via sgo.yaml.
package memgen

import (
	"embed"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// ImportPath is where the in-memory adapter lives, e.g.
// "<module>/internal/infrastructure/persistence/memory" (ARCHITECTURE.md
// §17).
func ImportPath(module string) string {
	return path.Join(module, "internal/infrastructure/persistence/memory")
}

// autoQuery is one repository_query method memgen can safely
// auto-implement directly in the generated (not owned) adapter file
// (ARCHITECTURE.md §21): a linear scan keyed on exactly one scalar
// parameter whose name case-insensitively matches an exported field on
// the domain entity.
type autoQuery struct {
	core.RepositoryQueryMethod
	FieldGoName string // the matched domain field, e.g. "Email"
}

// Generate writes
// internal/infrastructure/persistence/memory/<entity>_repository_gen.go,
// a trivial implementation of the aggregate's repository port
// (core.GenerateAggregateRepositoryPort's `<Entity>Repository` interface
// in internal/domain/<entity> — identical CRUD shape to the port this
// replaced, so this adapter's own Create/Get/List/Update/Delete methods
// don't need to change, only which domain package they operate on).
//
// Also implements any repository_query method (ARCHITECTURE.md §21)
// memgen can confidently map to a linear scan — see autoQuery — directly
// in this generated file; any it can't falls back to the same
// owned-stub-file pattern the real persistence engines use
// (GenerateRepositoryQueryStubs), rather than guessing at a mapping
// that could silently return wrong data.
func Generate(f *sgoproto.File, fd protoreflect.FileDescriptor, p core.Paths, destDir string) error {
	entity := entityTitle(p.Entity)

	aggregateMD, err := core.AggregateRoot(fd, p.Entity)
	if err != nil {
		return err
	}
	aggregate := f.FindMessage(string(aggregateMD.Name()))
	if aggregate == nil {
		return fmt.Errorf("memgen: aggregate root %s has no corresponding IR message (compiler/IR mismatch)", aggregateMD.Name())
	}

	queryMethods, err := core.RepositoryQueryMethods(fd, f, p.Entity)
	if err != nil {
		return err
	}
	autoMethods, stubMethods := splitAutoImplementable(queryMethods, aggregate)

	data := struct {
		Entity           string
		EntityLower      string
		DomainPkg        string
		DomainImportPath string
		AutoQueries      []autoQuery
	}{
		Entity:           entity,
		EntityLower:      p.Entity,
		DomainPkg:        p.Entity,
		DomainImportPath: p.AggregateDomainImportPath(),
		AutoQueries:      autoMethods,
	}

	destPath := filepath.Join(destDir, p.Entity+"_repository_gen.go")
	if err := gengo.Write(templatesFS, "templates/memory_repository_gen.go.tmpl", data, destPath); err != nil {
		return err
	}

	return core.GenerateRepositoryQueryStubs("memory", p.Entity, p.AggregateDomainImportPath(), entity, stubMethods, destDir, p.Entity+"_repository.go")
}

// splitAutoImplementable separates methods into what memgen can safely
// auto-implement as a linear scan (auto) and what it can't confidently
// map (stub) — more than one parameter, or no matching field.
func splitAutoImplementable(methods []core.RepositoryQueryMethod, aggregate *sgoproto.Message) (auto []autoQuery, stub []core.RepositoryQueryMethod) {
	fieldByLower := make(map[string]string, len(aggregate.Fields))
	for _, field := range aggregate.Fields {
		fieldByLower[strings.ToLower(field.GoName)] = field.GoName
	}

	for _, m := range methods {
		if len(m.Params) == 1 {
			if goName, ok := fieldByLower[strings.ToLower(m.Params[0].Field.GoName)]; ok {
				auto = append(auto, autoQuery{RepositoryQueryMethod: m, FieldGoName: goName})
				continue
			}
		}
		stub = append(stub, m)
	}
	return auto, stub
}

func entityTitle(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
