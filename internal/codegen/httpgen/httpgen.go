package httpgen

import (
	"embed"
	"fmt"
	"path"
	"path/filepath"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// framework bundles the two templates and the path-param syntax a given
// HTTP framework needs. Adding a framework means adding one entry here
// and its two template files — nothing else in this package changes,
// matching ARCHITECTURE.md §8.1's "adding a framework is additive"
// goal.
type framework struct {
	serverTemplate string
	routesTemplate string
	colonStyle     bool // true for gin/echo (":name"), false for chi ("{name}", Route's own native syntax)
}

var frameworks = map[config.HTTPFramework]framework{
	config.HTTPFrameworkGin: {
		serverTemplate: "templates/gin_server.go.tmpl",
		routesTemplate: "templates/gin_routes.go.tmpl",
		colonStyle:     true,
	},
	config.HTTPFrameworkEcho: {
		serverTemplate: "templates/echo_server.go.tmpl",
		routesTemplate: "templates/echo_routes.go.tmpl",
		colonStyle:     true,
	},
	config.HTTPFrameworkChi: {
		serverTemplate: "templates/chi_server.go.tmpl",
		routesTemplate: "templates/chi_routes.go.tmpl",
		colonStyle:     false,
	},
}

// ColonStyle reports whether fw uses ":name" path-param syntax (gin,
// echo) rather than "{name}" (chi, Route.PathTemplate's own native
// syntax) — the exact same lookup GenerateRoutes uses, exported so a
// caller building a Route's FullPath outside this package (e.g.
// `sgo list endpoints`, ARCHITECTURE.md §18) shows exactly what the
// generated adapter actually registers, not a second guess at the
// framework's syntax.
func ColonStyle(fw config.HTTPFramework) (bool, error) {
	def, ok := frameworks[fw]
	if !ok {
		return false, fmt.Errorf("httpgen: unsupported HTTP framework %q", fw)
	}
	return def.colonStyle, nil
}

// ImportPath is where the HTTP adapter for the given framework lives,
// e.g. "<module>/internal/infrastructure/transport/http/gin"
// (ARCHITECTURE.md §17).
func ImportPath(module string, fw config.HTTPFramework) string {
	return path.Join(module, "internal/infrastructure/transport/http", string(fw))
}

// GenerateServer writes the framework's server boilerplate
// (server_gen.go: a thin wrapper exposing the framework's native
// engine/router plus Start). It has no per-entity content, so it's safe
// to call with zero services registered yet (from `sgo init`) and to
// call again on every later `sgo generate code` run.
func GenerateServer(fw config.HTTPFramework, destDir string) error {
	def, ok := frameworks[fw]
	if !ok {
		return fmt.Errorf("httpgen: unsupported HTTP framework %q", fw)
	}

	return gengo.Write(templatesFS, def.serverTemplate, nil, filepath.Join(destDir, "server_gen.go"))
}

// GenerateRoutes writes this entity's route registrations, derived from
// f's service: an RPC with a `google.api.http` annotation uses it,
// falling back to the Create/Get/List/Update/Delete naming convention
// otherwise (ARCHITECTURE.md §8.1/§20). fd is f's underlying descriptor
// — BuildRoutes needs it to read the annotation and the service's
// `(sgo.base_path)` override.
func GenerateRoutes(fw config.HTTPFramework, fd protoreflect.FileDescriptor, f *sgoproto.File, p core.Paths, destDir string) error {
	def, ok := frameworks[fw]
	if !ok {
		return fmt.Errorf("httpgen: unsupported HTTP framework %q", fw)
	}

	data, err := buildRoutesData(fd, f, p, def.colonStyle)
	if err != nil {
		return err
	}
	routesPath := filepath.Join(destDir, p.Entity+"_routes_gen.go")

	return gengo.Write(templatesFS, def.routesTemplate, data, routesPath)
}
