// Package grpcgen generates the gRPC server adapter
// (internal/infrastructure/transport/grpc/<entity>_grpc_server_gen.go):
// a handler per RPC that converts the wire request to its
// application-layer Command/Query DTO (mapper.<Input>ToApp), calls the
// application service, and builds the wire Response envelope from the
// result (core.ClassifyResponse — the same derivation httpgen's routes
// use, since a gRPC handler needs the real wire.Output struct where
// httpgen only needs a JSON key).
package grpcgen

import (
	"embed"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// ImportPath is where the gRPC adapter lives, e.g.
// "<module>/internal/infrastructure/transport/grpc" (ARCHITECTURE.md
// §17).
func ImportPath(module string) string {
	return path.Join(module, "internal/infrastructure/transport/grpc")
}

// templateMethod is one RPC's view for the template — sgoproto.Method
// plus its derived ResponseShape, so the template doesn't need to call
// back into core itself.
type templateMethod struct {
	sgoproto.Method
	Resp core.ResponseShape
}

// Generate writes
// internal/infrastructure/transport/grpc/<entity>_grpc_server_gen.go.
func Generate(f *sgoproto.File, p core.Paths, destDir string) error {
	svc, ok := primaryService(f)
	if !ok {
		return fmt.Errorf("%s declares no service; sgo needs exactly one service per proto file", f.Path)
	}

	entity := entityTitle(p.Entity)

	methods := make([]templateMethod, len(svc.Methods))
	for i, m := range svc.Methods {
		methods[i] = templateMethod{
			Method: m,
			Resp:   core.ClassifyResponse(f, entity, m.Name, m.Output),
		}
	}

	data := struct {
		Entity                string
		EntityLower           string
		ApplicationImportPath string
		MapperImportPath      string
		WireImportPath        string
		Methods               []templateMethod
	}{
		Entity:                entity,
		EntityLower:           lower(entity),
		ApplicationImportPath: p.ApplicationImportPath(),
		MapperImportPath:      p.TransportMapperImportPath(),
		WireImportPath:        p.WireImportPath(),
		Methods:               methods,
	}

	destPath := filepath.Join(destDir, p.Entity+"_grpc_server_gen.go")
	return gengo.Write(templatesFS, "templates/grpc_server_gen.go.tmpl", data, destPath)
}

func primaryService(f *sgoproto.File) (*sgoproto.Service, bool) {
	if len(f.Services) == 0 {
		return nil, false
	}
	return &f.Services[0], true
}

func entityTitle(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func lower(name string) string {
	if name == "" {
		return name
	}
	return strings.ToLower(name[:1]) + name[1:]
}
