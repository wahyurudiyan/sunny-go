package httpgen

import (
	"embed"
	"fmt"
	"path"
	"path/filepath"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// framework bundles the two templates and the id-path-param syntax a
// given HTTP framework needs. Adding a framework means adding one entry
// here and its two template files — nothing else in this package
// changes, matching ARCHITECTURE.md §8.1's "adding a framework is
// additive" goal.
type framework struct {
	serverTemplate string
	routesTemplate string
	idPlaceholder  string
}

var frameworks = map[config.HTTPFramework]framework{
	config.HTTPFrameworkGin: {
		serverTemplate: "templates/gin_server.go.tmpl",
		routesTemplate: "templates/gin_routes.go.tmpl",
		idPlaceholder:  ":id",
	},
	config.HTTPFrameworkEcho: {
		serverTemplate: "templates/echo_server.go.tmpl",
		routesTemplate: "templates/echo_routes.go.tmpl",
		idPlaceholder:  ":id",
	},
	config.HTTPFrameworkChi: {
		serverTemplate: "templates/chi_server.go.tmpl",
		routesTemplate: "templates/chi_routes.go.tmpl",
		idPlaceholder:  "{id}",
	},
}

// IDPlaceholder returns fw's id-path-param syntax (":id" for gin/echo,
// "{id}" for chi) — the exact same lookup GenerateRoutes uses, exported
// so a caller building a Route's FullPath outside this package (e.g.
// `sgo list endpoints`, ARCHITECTURE.md §18) shows exactly what the
// generated adapter actually registers, not a second guess at the
// framework's syntax.
func IDPlaceholder(fw config.HTTPFramework) (string, error) {
	def, ok := frameworks[fw]
	if !ok {
		return "", fmt.Errorf("httpgen: unsupported HTTP framework %q", fw)
	}
	return def.idPlaceholder, nil
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
// f's service per ARCHITECTURE.md §4/§12 (no google.api.http support
// yet — routes come from the Create/Get/List/Update/Delete naming
// convention).
func GenerateRoutes(fw config.HTTPFramework, f *sgoproto.File, p core.Paths, destDir string) error {
	def, ok := frameworks[fw]
	if !ok {
		return fmt.Errorf("httpgen: unsupported HTTP framework %q", fw)
	}

	data := buildRoutesData(f, p, def.idPlaceholder)
	routesPath := filepath.Join(destDir, p.Entity+"_routes_gen.go")

	return gengo.Write(templatesFS, def.routesTemplate, data, routesPath)
}
