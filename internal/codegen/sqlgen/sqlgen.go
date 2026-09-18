// Package sqlgen generates the Postgres/MySQL persistence adapter for
// an entity's repository port (ARCHITECTURE.md §8.2), in either
// self-managed (database/sql + hand-written SQL) or ORM (GORM) mode,
// selected by sgo.yaml's persistence.mode. Both engines share the same
// two templates; only driver/dialect-specific strings differ.
package sqlgen

import (
	"embed"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

type engineDef struct {
	driverImport    string // self-managed driver, blank-imported for side effects
	sqlDriver       string // name passed to sql.Open
	dsnEnvPrefix    string // e.g. "POSTGRES"
	placeholder     func(i int) string
	gormImport      string // ORM mode dialector package
	gormOpen        string // e.g. "postgres.Open"
	dsnStyle        string // "postgres" | "mysql" — how Connect() builds the DSN string
	defaultPort     string
	defaultUser     string
	defaultPassword string
	defaultDB       string
}

var engines = map[config.PersistenceEngine]engineDef{
	config.PersistenceEnginePostgres: {
		driverImport:    "github.com/jackc/pgx/v5/stdlib",
		sqlDriver:       "pgx",
		dsnEnvPrefix:    "POSTGRES",
		placeholder:     func(i int) string { return fmt.Sprintf("$%d", i) },
		gormImport:      "gorm.io/driver/postgres",
		gormOpen:        "postgres.Open",
		dsnStyle:        "postgres",
		defaultPort:     "5432",
		defaultUser:     "postgres",
		defaultPassword: "postgres",
		defaultDB:       "postgres",
	},
	config.PersistenceEngineMySQL: {
		driverImport:    "github.com/go-sql-driver/mysql",
		sqlDriver:       "mysql",
		dsnEnvPrefix:    "MYSQL",
		placeholder:     func(int) string { return "?" },
		gormImport:      "gorm.io/driver/mysql",
		gormOpen:        "mysql.Open",
		dsnStyle:        "mysql",
		defaultPort:     "3306",
		defaultUser:     "root",
		defaultPassword: "root",
		defaultDB:       "app",
	},
}

// Supports reports whether engine is one sqlgen can generate for.
func Supports(engine config.PersistenceEngine) bool {
	_, ok := engines[engine]
	return ok
}

// ImportPath is where engine's persistence adapter lives, e.g.
// "<module>/internal/adapter/out/persistence/postgres".
func ImportPath(module string, engine config.PersistenceEngine) string {
	return path.Join(module, "internal/adapter/out/persistence", string(engine))
}

// column is one entity field persisted as a SQL column (or a GORM
// struct field). Only top-level scalar fields are supported — a message
// or repeated field is skipped, since neither raw SQL columns nor a
// flat GORM model represent nested structures without a schema decision
// this generator doesn't make. See ARCHITECTURE.md §12.
type column struct {
	GoName string
	SQL    string // snake_case column name
	Field  sgoproto.Field
}

func scalarColumns(msg *sgoproto.Message) []column {
	var cols []column
	for _, f := range msg.Fields {
		if f.GoName == "Id" || f.IsMessage() || f.Repeated {
			continue
		}
		cols = append(cols, column{GoName: f.GoName, SQL: f.Name, Field: f})
	}
	return cols
}

// Generate writes the entity's repository adapter for engine, in the
// mode sgo.yaml currently selects.
func Generate(engine config.PersistenceEngine, mode config.PersistenceMode, f *sgoproto.File, p core.Paths, destDir string) error {
	def, ok := engines[engine]
	if !ok {
		return fmt.Errorf("sqlgen: unsupported SQL engine %q", engine)
	}

	msg := f.FindMessage(entityTitle(p.Entity))
	if msg == nil {
		return fmt.Errorf("sqlgen: proto file has no %s message", entityTitle(p.Entity))
	}
	cols := scalarColumns(msg)

	if mode == config.PersistenceModeORM {
		return generateORM(def, engine, cols, p, destDir)
	}
	return generateSelfManaged(def, engine, cols, p, destDir)
}

func entityTitle(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func table(entity string) string {
	return strings.ToLower(entity) + "s"
}

func destPath(destDir, entity, suffix string) string {
	return filepath.Join(destDir, strings.ToLower(entity)+suffix)
}
