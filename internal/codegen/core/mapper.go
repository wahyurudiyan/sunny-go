package core

import (
	"path/filepath"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// GenerateMapper writes internal/adapter/mapper/<entity>_mapper_gen.go:
// ToDomain/FromDomain conversions between contract/gen's wire types and
// core/domain's hand-shaped structs, for every message in f. This is the
// anti-corruption layer described in ARCHITECTURE.md §5 — it lives in
// the adapter layer, not core, since core must stay free of any
// dependency on the generated protobuf types.
func GenerateMapper(f *sgoproto.File, p Paths, destDir string) error {
	data := struct {
		WireImportPath   string
		DomainImportPath string
		Messages         []sgoproto.Message
	}{
		WireImportPath:   p.WireImportPath(),
		DomainImportPath: p.DomainImportPath(),
		Messages:         f.Messages,
	}

	path := filepath.Join(destDir, p.Entity+"_mapper_gen.go")
	return writeGoFile("templates/mapper_gen.go.tmpl", data, path)
}
