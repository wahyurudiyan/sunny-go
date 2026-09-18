// Package bootstrap generates internal/bootstrap/wire_gen.go, the
// composition root that wires every generated service to the in-memory
// repository (memgen) and starts both the HTTP (httpgen) and gRPC
// (grpcgen) servers. It's regenerated on every `sgo generate code` run
// so it always reflects the full current set of services — not just the
// one just generated.
package bootstrap

import (
	"embed"
	"path"
	"path/filepath"
	"strings"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/grpcgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/httpgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/memgen"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

var engineFields = map[config.HTTPFramework]string{
	config.HTTPFrameworkGin:  "Engine",
	config.HTTPFrameworkEcho: "Echo",
	config.HTTPFrameworkChi:  "Router",
}

type entityData struct {
	Lower string
	Title string
}

// Generate writes internal/bootstrap/wire_gen.go, wiring every entity in
// services.
func Generate(module string, fw config.HTTPFramework, services []string, destDir string) error {
	entities := make([]entityData, 0, len(services))
	for _, s := range services {
		entities = append(entities, entityData{Lower: s, Title: entityTitle(s)})
	}

	data := struct {
		HTTPImportPath    string
		GRPCImportPath    string
		MemoryImportPath  string
		ServiceImportPath string
		EngineField       string
		Entities          []entityData
	}{
		HTTPImportPath:    httpgen.ImportPath(module, fw),
		GRPCImportPath:    grpcgen.ImportPath(module),
		MemoryImportPath:  memgen.ImportPath(module),
		ServiceImportPath: ServiceImportPath(module),
		EngineField:       engineFields[fw],
		Entities:          entities,
	}

	destPath := filepath.Join(destDir, "wire_gen.go")
	return gengo.Write(templatesFS, "templates/wire_gen.go.tmpl", data, destPath)
}

// ServiceImportPath is where every entity's service implementation
// lives (they share one Go package — see internal/codegen/core), e.g.
// "<module>/internal/core/service".
func ServiceImportPath(module string) string {
	return path.Join(module, "internal/core/service")
}

func entityTitle(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
