package core

import (
	"fmt"
	"os"
	"path/filepath"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// GenerateService writes internal/core/service/<entity>_service.go.
//
// If the file doesn't exist yet, it's created in full: struct,
// constructor, and one panic-stub method per usecase method. If it
// already exists, it is never overwritten — see ensureGoMethods
// (astmethods.go), which only ever appends stubs for methods the file
// is missing. This is the safe-regeneration guarantee from
// ARCHITECTURE.md §6: a second `sgo generate code` run must never
// remove or alter business logic a developer already wrote into this
// file.
func GenerateService(f *sgoproto.File, p Paths, destDir string) error {
	svc, ok := primaryService(f)
	if !ok {
		return fmt.Errorf("%s declares no service; sgo needs exactly one service per proto file", f.Path)
	}

	receiver := entityTitle(p.Entity) + "Service"
	path := filepath.Join(destDir, p.Entity+"_service.go")

	_, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		return writeServiceSkeleton(path, p, receiver, svc.Methods)
	case err != nil:
		return fmt.Errorf("failed to check %s: %w", path, err)
	default:
		return ensureMethods(path, p, receiver, svc.Methods)
	}
}

func writeServiceSkeleton(path string, p Paths, receiver string, methods []sgoproto.Method) error {
	data := struct {
		Entity            string
		DomainPkg         string
		DomainImportPath  string
		PortOutImportPath string
		Methods           []sgoproto.Method
	}{
		Entity:            entityTitle(p.Entity),
		DomainPkg:         p.Entity,
		DomainImportPath:  p.DomainImportPath(),
		PortOutImportPath: p.PortOutImportPath(),
		Methods:           methods,
	}

	return writeGoFile("templates/service_owned.go.tmpl", data, path)
}

// ensureMethods adapts ensureGoMethods (astmethods.go) to this
// generator's stub shape: `req *<DomainPkg>.<Input>` /
// `*<DomainPkg>.<Output>`, matching the pre-Phase-12 usecase-port
// service convention.
func ensureMethods(path string, p Paths, receiver string, methods []sgoproto.Method) error {
	byName := make(map[string]sgoproto.Method, len(methods))
	names := make([]string, len(methods))
	for i, m := range methods {
		byName[m.Name] = m
		names[i] = m.Name
	}

	return ensureGoMethods(path, receiver, p.Entity, names, func(name string) ([]byte, error) {
		m := byName[name]
		return renderTemplate("templates/service_method_stub.go.tmpl", struct {
			Receiver  string
			DomainPkg string
			Name      string
			Input     string
			Output    string
		}{
			Receiver:  receiver,
			DomainPkg: p.Entity,
			Name:      m.Name,
			Input:     m.Input,
			Output:    m.Output,
		})
	})
}
