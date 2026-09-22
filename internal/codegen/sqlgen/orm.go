package sqlgen

import (
	"path/filepath"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

type ormData struct {
	Package          string
	Entity           string
	EntityLower      string
	DomainPkg        string
	DomainImportPath string
	GormImport       string
	GormOpen         string
	DSNEnvPrefix     string
	DSNStyle         string
	DefaultPort      string
	DefaultUser      string
	DefaultPassword  string
	DefaultDB        string
	Table            string
	Columns          []column
}

func generateORM(def engineDef, engine config.PersistenceEngine, cols []column, p core.Paths, destDir string) error {
	data := ormData{
		Package:          string(engine),
		Entity:           entityTitle(p.Entity),
		EntityLower:      p.Entity,
		DomainPkg:        p.Entity,
		DomainImportPath: p.AggregateDomainImportPath(),
		GormImport:       def.gormImport,
		GormOpen:         def.gormOpen,
		DSNEnvPrefix:     def.dsnEnvPrefix,
		DSNStyle:         def.dsnStyle,
		DefaultPort:      def.defaultPort,
		DefaultUser:      def.defaultUser,
		DefaultPassword:  def.defaultPassword,
		DefaultDB:        def.defaultDB,
		Table:            table(p.Entity),
		Columns:          cols,
	}

	if err := gengo.Write(templatesFS, "templates/sql_orm_repository_gen.go.tmpl", data, destPath(destDir, p.Entity, "_repository_gen.go")); err != nil {
		return err
	}

	return gengo.Write(templatesFS, "templates/sql_orm_conn_gen.go.tmpl", data, filepath.Join(destDir, "conn_gen.go"))
}
