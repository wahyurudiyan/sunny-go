// Package searchgen generates the project-scoped (not per-entity)
// search port and its Elasticsearch adapter (ARCHITECTURE.md §8.2/§8.3).
// Like cachegen, this is available infrastructure once generated, but
// nothing wires it into a service automatically — that would mean
// changing an owned file's constructor signature.
package searchgen

import (
	"embed"
	"path"
	"path/filepath"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// ImportPath is where the Elasticsearch adapter lives, e.g.
// "<module>/internal/adapter/out/search/elasticsearch".
func ImportPath(module string) string {
	return path.Join(module, "internal/adapter/out/search/elasticsearch")
}

// GeneratePort writes internal/core/port/out/search.go.
func GeneratePort(destDir string) error {
	return gengo.Write(templatesFS, "templates/search_port_gen.go.tmpl", nil, filepath.Join(destDir, "search.go"))
}

// GenerateElasticsearch writes the Elasticsearch adapter into destDir.
func GenerateElasticsearch(module, destDir string) error {
	data := struct{ PortOutImportPath string }{PortOutImportPath: path.Join(module, "internal/core/port/out")}

	if err := gengo.Write(templatesFS, "templates/elasticsearch_search_gen.go.tmpl", data, filepath.Join(destDir, "search_gen.go")); err != nil {
		return err
	}
	return gengo.Write(templatesFS, "templates/elasticsearch_conn_gen.go.tmpl", nil, filepath.Join(destDir, "conn_gen.go"))
}
