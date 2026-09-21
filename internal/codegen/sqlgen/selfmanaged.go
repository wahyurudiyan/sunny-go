package sqlgen

import (
	"path/filepath"
	"strings"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

type selfManagedData struct {
	Package          string
	Entity           string
	DomainPkg        string
	DomainImportPath string
	DriverImport     string
	SQLDriver        string
	DSNEnvPrefix     string
	DSNStyle         string
	DefaultPort      string
	DefaultUser      string
	DefaultPassword  string
	DefaultDB        string
	Table            string
	Columns          []column
	InsertQuery      string
	SelectQuery      string
	ListQuery        string
	ListQueryLimited string
	CountQuery       string
	UpdateQuery      string
	DeleteQuery      string
	MigrateQuery     string
}

func generateSelfManaged(def engineDef, engine config.PersistenceEngine, cols []column, p core.Paths, destDir string) error {
	table := table(p.Entity)

	insertCols := append([]string{"id"}, columnNames(cols)...)
	insertPlaceholders := placeholders(def, 1, len(insertCols))
	selectCols := strings.Join(append([]string{"id"}, columnNames(cols)...), ", ")

	setClauses := make([]string, len(cols))
	for i, c := range cols {
		setClauses[i] = c.SQL + " = " + def.placeholder(i+1)
	}

	migrateCols := make([]string, len(cols))
	for i, c := range cols {
		migrateCols[i] = c.SQL + " TEXT"
	}
	migrateQuery := "CREATE TABLE IF NOT EXISTS " + table + " (id VARCHAR(255) PRIMARY KEY"
	if len(migrateCols) > 0 {
		migrateQuery += ", " + strings.Join(migrateCols, ", ")
	}
	migrateQuery += ")"

	data := selfManagedData{
		Package:          string(engine),
		Entity:           entityTitle(p.Entity),
		DomainPkg:        p.Entity,
		DomainImportPath: p.DomainImportPath(),
		DriverImport:     def.driverImport,
		SQLDriver:        def.sqlDriver,
		DSNEnvPrefix:     def.dsnEnvPrefix,
		DSNStyle:         def.dsnStyle,
		DefaultPort:      def.defaultPort,
		DefaultUser:      def.defaultUser,
		DefaultPassword:  def.defaultPassword,
		DefaultDB:        def.defaultDB,
		Table:            table,
		Columns:          cols,
		InsertQuery:      "INSERT INTO " + table + " (" + strings.Join(insertCols, ", ") + ") VALUES (" + strings.Join(insertPlaceholders, ", ") + ")",
		SelectQuery:      "SELECT " + selectCols + " FROM " + table + " WHERE id = " + def.placeholder(1),
		ListQuery:        "SELECT " + selectCols + " FROM " + table + " ORDER BY id",
		ListQueryLimited: "SELECT " + selectCols + " FROM " + table + " ORDER BY id LIMIT " + def.placeholder(1) + " OFFSET " + def.placeholder(2),
		CountQuery:       "SELECT COUNT(*) FROM " + table,
		UpdateQuery:      "UPDATE " + table + " SET " + strings.Join(setClauses, ", ") + " WHERE id = " + def.placeholder(len(cols)+1),
		DeleteQuery:      "DELETE FROM " + table + " WHERE id = " + def.placeholder(1),
		MigrateQuery:     migrateQuery,
	}

	if err := gengo.Write(templatesFS, "templates/sql_selfmanaged_repository_gen.go.tmpl", data, destPath(destDir, p.Entity, "_repository_gen.go")); err != nil {
		return err
	}

	return gengo.Write(templatesFS, "templates/sql_selfmanaged_conn_gen.go.tmpl", data, filepath.Join(destDir, "conn_gen.go"))
}

func columnNames(cols []column) []string {
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.SQL
	}
	return names
}

func placeholders(def engineDef, start, count int) []string {
	out := make([]string, count)
	for i := 0; i < count; i++ {
		out[i] = def.placeholder(start + i)
	}
	return out
}
