package core

import (
	"fmt"
	"os"
	"path/filepath"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// GenerateDomain writes internal/core/domain/<entity>/<entity>_gen.go
// (every message in f, always overwritten) and, if it doesn't already
// exist, <entity>.go (owned, for hand-written domain methods).
func GenerateDomain(f *sgoproto.File, p Paths, destDir string) error {
	genPath := filepath.Join(destDir, p.Entity+"_gen.go")
	genData := struct {
		Package  string
		Messages []sgoproto.Message
	}{
		Package:  p.Entity,
		Messages: f.Messages,
	}
	if err := writeGoFile("templates/domain_gen.go.tmpl", genData, genPath); err != nil {
		return err
	}

	ownedPath := filepath.Join(destDir, p.Entity+".go")
	if _, err := os.Stat(ownedPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check %s: %w", ownedPath, err)
	}

	ownedData := struct {
		Package string
		Entity  string
	}{
		Package: p.Entity,
		Entity:  entityTitle(p.Entity),
	}

	return writeGoFile("templates/domain_owned.go.tmpl", ownedData, ownedPath)
}
