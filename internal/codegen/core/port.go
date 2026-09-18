package core

import (
	"fmt"
	"path/filepath"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// GenerateUsecasePort writes internal/core/port/in/<entity>_usecase.go:
// one interface method per RPC on f's (single) service.
func GenerateUsecasePort(f *sgoproto.File, p Paths, destDir string) error {
	svc, ok := primaryService(f)
	if !ok {
		return fmt.Errorf("%s declares no service; sgo needs exactly one service per proto file", f.Path)
	}

	data := struct {
		Entity           string
		DomainPkg        string
		DomainImportPath string
		Methods          []sgoproto.Method
	}{
		Entity:           entityTitle(p.Entity),
		DomainPkg:        p.Entity,
		DomainImportPath: p.DomainImportPath(),
		Methods:          svc.Methods,
	}

	path := filepath.Join(destDir, p.Entity+"_usecase.go")
	return writeGoFile("templates/usecase_port.go.tmpl", data, path)
}

// GenerateRepositoryPort writes internal/core/port/out/<entity>_repository.go:
// the fixed CRUD shape described in ARCHITECTURE.md §8.2, independent of
// the proto service's exact RPC names.
func GenerateRepositoryPort(p Paths, destDir string) error {
	data := struct {
		Entity           string
		DomainPkg        string
		DomainImportPath string
	}{
		Entity:           entityTitle(p.Entity),
		DomainPkg:        p.Entity,
		DomainImportPath: p.DomainImportPath(),
	}

	path := filepath.Join(destDir, p.Entity+"_repository.go")
	return writeGoFile("templates/repository_port.go.tmpl", data, path)
}
