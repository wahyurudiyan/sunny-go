// Package bootstrap generates
// internal/infrastructure/bootstrap/wire_gen.go, the composition root
// that wires every generated service — repository, a shared no-op
// EventPublisher, and the application service itself
// (ARCHITECTURE.md §17) — to the repository adapter sgo.yaml's
// persistence selection names (falling back to the in-memory default —
// Decision #15), and starts both the HTTP (httpgen) and gRPC (grpcgen)
// servers. It's regenerated on every `sgo generate code` run so it
// always reflects the full current set of services and the project's
// current selections — not just the one entity just generated.
package bootstrap

import (
	"embed"
	"path/filepath"
	"strings"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/cachegen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/grpcgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/httpgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/memgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/mongogen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/searchgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/sqlgen"
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
	Lower                 string
	Title                 string
	ApplicationImportPath string
}

// Generate writes internal/infrastructure/bootstrap/wire_gen.go, wiring
// every service in cfg.Services to the repository adapter
// cfg.Persistence names (or the in-memory default if none is selected)
// and a shared no-op EventPublisher, plus a cache/search client if
// cfg.Cache/cfg.Search name one — see ARCHITECTURE.md §8.2/§8.3/§17.
func Generate(cfg *config.Config, destDir string) error {
	entities := make([]entityData, 0, len(cfg.Services))
	for _, s := range cfg.Services {
		entities = append(entities, entityData{
			Lower:                 s,
			Title:                 entityTitle(s),
			ApplicationImportPath: core.Paths{Module: cfg.Module, Entity: s}.ApplicationImportPath(),
		})
	}

	persistence := resolvePersistence(cfg)

	data := struct {
		HTTPImportPath       string
		GRPCImportPath       string
		ApplicationPortsPath string
		EngineField          string
		Entities             []entityData

		RepoPackage    string
		RepoImportPath string
		ConnArg        string
		HasConnect     bool
		ConnectExpr    string
		MigrateExpr    string

		UsesCache        bool
		CacheImportPath  string
		UsesSearch       bool
		SearchImportPath string
	}{
		HTTPImportPath:       httpgen.ImportPath(cfg.Module, cfg.HTTPFramework),
		GRPCImportPath:       grpcgen.ImportPath(cfg.Module),
		ApplicationPortsPath: core.Paths{Module: cfg.Module}.ApplicationPortsImportPath(),
		EngineField:          engineFields[cfg.HTTPFramework],
		Entities:             entities,

		RepoPackage:    persistence.pkg,
		RepoImportPath: persistence.importPath,
		ConnArg:        persistence.connArg,
		HasConnect:     persistence.connectExpr != "",
		ConnectExpr:    persistence.connectExpr,
		MigrateExpr:    persistence.migrateExpr,
	}

	if hasEngine(cfg.Cache, config.CacheEngineRedis) {
		data.UsesCache = true
		data.CacheImportPath = cachegen.ImportPath(cfg.Module)
	}
	if hasEngine(cfg.Search, config.SearchEngineElasticsearch) {
		data.UsesSearch = true
		data.SearchImportPath = searchgen.ImportPath(cfg.Module)
	}

	destPath := filepath.Join(destDir, "wire_gen.go")
	return gengo.Write(templatesFS, "templates/wire_gen.go.tmpl", data, destPath)
}

type persistenceChoice struct {
	pkg         string // Go package identifier, e.g. "postgres", "memory"
	importPath  string
	connArg     string // argument to New<Entity>Repository(...); "" for memory
	connectExpr string // e.g. "postgres.Connect(ctx)"; "" for memory (no connection needed)
	migrateExpr string // e.g. "postgres.AutoMigrate(ctx, db)"; "" if not applicable
}

// resolvePersistence picks the first persistence engine cfg selected, in
// the mode cfg.Persistence.Mode names, or falls back to the in-memory
// default if none was selected.
func resolvePersistence(cfg *config.Config) persistenceChoice {
	if len(cfg.Persistence.Engines) == 0 {
		return persistenceChoice{pkg: "memory", importPath: memgen.ImportPath(cfg.Module)}
	}

	engine := cfg.Persistence.Engines[0]

	switch engine {
	case config.PersistenceEnginePostgres, config.PersistenceEngineMySQL:
		importPath := sqlgen.ImportPath(cfg.Module, engine)
		pkg := string(engine)
		if cfg.Persistence.Mode == config.PersistenceModeORM {
			return persistenceChoice{
				pkg: pkg, importPath: importPath, connArg: "db",
				connectExpr: pkg + ".Connect()",
				migrateExpr: pkg + ".AutoMigrate(db)",
			}
		}
		return persistenceChoice{
			pkg: pkg, importPath: importPath, connArg: "db",
			connectExpr: pkg + ".Connect(ctx)",
			migrateExpr: pkg + ".AutoMigrate(ctx, db)",
		}
	case config.PersistenceEngineMongo:
		importPath := mongogen.ImportPath(cfg.Module)
		return persistenceChoice{
			pkg: "mongo", importPath: importPath, connArg: "db",
			connectExpr: "mongo.Connect(ctx)",
		}
	default:
		return persistenceChoice{pkg: "memory", importPath: memgen.ImportPath(cfg.Module)}
	}
}

func hasEngine[T comparable](engines []T, want T) bool {
	for _, e := range engines {
		if e == want {
			return true
		}
	}
	return false
}

func entityTitle(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
