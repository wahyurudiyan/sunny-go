package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateRepositoryQueryStubs writes (creating once, then only
// appending any newly added one) destDir/filename: an owned companion
// file to a persistence adapter's generated
// <entity>_repository_gen.go, one panic("sgo: TODO implement <Method>")
// stub per method in methods (ARCHITECTURE.md §21/Phase 16) — sgo has
// no way to know what query logic a real engine's custom method needs,
// so it stubs the signature and leaves the body to you. Uses the exact
// append-only-new-stubs mechanism (Decision #12)
// internal/application/<entity>/service.go already relies on, applied
// one layer lower: a hand-written implementation survives every later
// `sgo generate code` run, and a newly added custom method gets a
// fresh stub appended without touching what's already there. Writes
// nothing, and creates no file, if methods is empty — a project with no
// repository_query RPCs (or, for memgen's caller, none it couldn't
// already auto-implement) gets no owned file cluttering its adapter
// package.
func GenerateRepositoryQueryStubs(pkg, domainPkg, domainImportPath, aggregateName string, methods []RepositoryQueryMethod, destDir, filename string) error {
	if len(methods) == 0 {
		return nil
	}

	receiver := aggregateName + "Repository"
	path := filepath.Join(destDir, filename)

	names := make([]string, len(methods))
	byName := make(map[string]RepositoryQueryMethod, len(methods))
	for i, m := range methods {
		names[i] = m.Name
		byName[m.Name] = m
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return writeRepositoryQueryStubSkeleton(path, pkg, domainPkg, domainImportPath, aggregateName, methods)
	} else if err != nil {
		return fmt.Errorf("failed to check %s: %w", path, err)
	}

	return ensureGoMethods(path, receiver, names, func(name string) ([]byte, error) {
		m := byName[name]
		return renderTemplate("templates/repository_query_stub.go.tmpl", struct {
			Receiver         string
			AggregateName    string
			DomainPkg        string
			DomainImportPath string
			Method           RepositoryQueryMethod
		}{
			Receiver:         receiver,
			AggregateName:    aggregateName,
			DomainPkg:        domainPkg,
			DomainImportPath: domainImportPath,
			Method:           m,
		})
	})
}

func writeRepositoryQueryStubSkeleton(path, pkg, domainPkg, domainImportPath, aggregateName string, methods []RepositoryQueryMethod) error {
	data := struct {
		Package          string
		AggregateName    string
		DomainPkg        string
		DomainImportPath string
		Methods          []RepositoryQueryMethod
	}{
		Package:          pkg,
		AggregateName:    aggregateName,
		DomainPkg:        domainPkg,
		DomainImportPath: domainImportPath,
		Methods:          methods,
	}
	return writeGoFile("templates/repository_query_stubs_owned.go.tmpl", data, path)
}
