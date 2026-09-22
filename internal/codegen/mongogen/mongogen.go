// Package mongogen generates the MongoDB persistence adapter for an
// entity's repository port (ARCHITECTURE.md §8.2), using the official
// go.mongodb.org/mongo-driver. There's no self-managed/ORM split for
// Mongo — a document store doesn't have the "raw SQL vs ORM" distinction
// SQL engines do.
package mongogen

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

// ImportPath is where the Mongo adapter lives, e.g.
// "<module>/internal/infrastructure/persistence/mongo"
// (ARCHITECTURE.md §17).
func ImportPath(module string) string {
	return path.Join(module, "internal/infrastructure/persistence/mongo")
}

type column struct {
	GoName string
	BSON   string
	Field  sgoproto.Field
}

// Generate writes the entity's Mongo repository adapter and (once per
// project, harmlessly re-written on every call) the connection helper,
// plus — for every RPC marked `option (sgo.repository_query) = true;`
// (ARCHITECTURE.md §21) — an owned companion file with one
// panic("sgo: TODO implement ...") stub per custom method: unlike
// memgen, a real engine's query logic can't be auto-generated, since
// sgo has no way to know what Mongo query a method like FindByEmail
// needs.
func Generate(f *sgoproto.File, fd protoreflect.FileDescriptor, p core.Paths, destDir string) error {
	entity := entityTitle(p.Entity)
	msg := f.FindMessage(entity)
	if msg == nil {
		return fmt.Errorf("mongogen: proto file has no %s message", entity)
	}

	var cols []column
	for _, field := range msg.Fields {
		if field.GoName == "Id" || field.IsMessage() || field.Repeated {
			continue
		}
		cols = append(cols, column{GoName: field.GoName, BSON: field.Name, Field: field})
	}

	data := struct {
		Entity           string
		EntityLower      string
		DomainPkg        string
		DomainImportPath string
		Collection       string
		Columns          []column
	}{
		Entity:           entityTitle(p.Entity),
		EntityLower:      p.Entity,
		DomainPkg:        p.Entity,
		DomainImportPath: p.AggregateDomainImportPath(),
		Collection:       strings.ToLower(p.Entity) + "s",
		Columns:          cols,
	}

	repoPath := filepath.Join(destDir, p.Entity+"_repository_gen.go")
	if err := gengo.Write(templatesFS, "templates/mongo_repository_gen.go.tmpl", data, repoPath); err != nil {
		return err
	}

	if err := gengo.Write(templatesFS, "templates/mongo_conn_gen.go.tmpl", data, filepath.Join(destDir, "conn_gen.go")); err != nil {
		return err
	}

	queryMethods, err := core.RepositoryQueryMethods(fd, f, p.Entity)
	if err != nil {
		return err
	}
	return core.GenerateRepositoryQueryStubs("mongo", p.Entity, p.AggregateDomainImportPath(), entity, queryMethods, destDir, p.Entity+"_repository.go")
}

func entityTitle(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
