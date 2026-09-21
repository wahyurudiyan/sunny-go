// Package grpcgen generates the gRPC server adapter
// (internal/adapter/in/grpc/<entity>_grpc_server_gen.go): a handler per
// RPC that converts wire types to domain types via the mapper, calls the
// usecase port, and converts the result back.
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
// "<module>/internal/adapter/in/grpc".
func ImportPath(module string) string {
	return path.Join(module, "internal/adapter/in/grpc")
}

// Generate writes internal/adapter/in/grpc/<entity>_grpc_server_gen.go.
func Generate(f *sgoproto.File, p core.Paths, destDir string) error {
	svc, ok := primaryService(f)
	if !ok {
		return fmt.Errorf("%s declares no service; sgo needs exactly one service per proto file", f.Path)
	}

	entity := entityTitle(p.Entity)

	data := struct {
		Entity           string
		EntityLower      string
		PortInImportPath string
		MapperImportPath string
		WireImportPath   string
		Methods          []sgoproto.Method
	}{
		Entity:           entity,
		EntityLower:      lower(entity),
		PortInImportPath: p.PortInImportPath(),
		MapperImportPath: p.MapperImportPath(),
		WireImportPath:   p.WireImportPath(),
		Methods:          svc.Methods,
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
