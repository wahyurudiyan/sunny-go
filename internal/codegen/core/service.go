package core

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// GenerateService writes internal/core/service/<entity>_service.go.
//
// If the file doesn't exist yet, it's created in full: struct,
// constructor, and one panic-stub method per usecase method. If it
// already exists, it is never overwritten — see ensureMethods, which
// only ever appends stubs for methods the file is missing. This is the
// safe-regeneration guarantee from ARCHITECTURE.md §6: a second
// `sgo generate code` run must never remove or alter business logic a
// developer already wrote into this file.
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

// ensureMethods makes sure path (an existing owned service file) has a
// method for every entry in methods, appending a panic-stub for any
// that's missing. Every existing method — whatever its body — is left
// completely untouched: this function never rewrites or removes a line
// that was already there. An existing exported method on the receiver
// that ISN'T in methods (the usecase interface no longer declares it)
// gets a one-line warning comment inserted above it, once, rather than
// being deleted.
func ensureMethods(path string, p Paths, receiver string, methods []sgoproto.Method) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse existing %s: %w", path, err)
	}

	required := make(map[string]bool, len(methods))
	for _, m := range methods {
		required[m.Name] = true
	}

	existing := map[string]bool{}
	type orphan struct {
		Line int
		Name string
	}
	var orphans []orphan

	for _, decl := range astFile.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || receiverTypeName(fn.Recv) != receiver {
			continue
		}

		existing[fn.Name.Name] = true

		if fn.Name.IsExported() && !required[fn.Name.Name] {
			orphans = append(orphans, orphan{Line: fset.Position(fn.Pos()).Line, Name: fn.Name.Name})
		}
	}

	lines := strings.Split(string(src), "\n")

	sort.Slice(orphans, func(i, j int) bool { return orphans[i].Line > orphans[j].Line })
	for _, o := range orphans {
		comment := orphanComment(o.Name, p.Entity)
		idx := o.Line - 1
		if idx > 0 && strings.TrimSpace(lines[idx-1]) == comment {
			continue // already annotated by a previous run
		}
		lines = append(lines[:idx], append([]string{comment}, lines[idx:]...)...)
	}

	result := strings.Join(lines, "\n")

	for _, m := range methods {
		if existing[m.Name] {
			continue
		}

		stub, err := renderTemplate("templates/service_method_stub.go.tmpl", struct {
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
		if err != nil {
			return err
		}

		result += "\n" + string(stub)
	}

	formatted, err := format.Source([]byte(result))
	if err != nil {
		return fmt.Errorf("failed to gofmt %s after regenerating: %w", path, err)
	}

	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

func orphanComment(method, entity string) string {
	return fmt.Sprintf("// sgo: %s is no longer part of %sUseCase; remove if unused", method, entityTitle(entity))
}

// receiverTypeName returns the bare type name of a method's receiver
// ("Foo" for both "f *Foo" and "f Foo"), or "" if recv doesn't describe
// a simple named-type receiver.
func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	expr := recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}
