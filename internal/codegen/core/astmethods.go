package core

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
)

// ensureGoMethods makes sure the existing Go file at path has a method
// for every name in methodNames, appending stubFor(name)'s rendered
// stub for any that's missing. Every existing method on receiver —
// whatever its body — is left completely untouched: this never
// rewrites or removes a line that was already there (ARCHITECTURE.md
// §6's safe-regeneration guarantee, Decision #12). An existing exported
// method on receiver that ISN'T in methodNames gets a one-line warning
// comment inserted above it, once, rather than being deleted.
//
// Shared by the pre-Phase-12 usecase-port service generator
// (service.go) and the Phase-12 application service generator
// (application.go) — the AST mechanism doesn't care what a stub's
// content is, only that methods exist.
func ensureGoMethods(path, receiver, entity string, methodNames []string, stubFor func(name string) ([]byte, error)) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse existing %s: %w", path, err)
	}

	required := make(map[string]bool, len(methodNames))
	for _, name := range methodNames {
		required[name] = true
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
		comment := orphanComment(o.Name, entity)
		idx := o.Line - 1
		if idx > 0 && strings.TrimSpace(lines[idx-1]) == comment {
			continue // already annotated by a previous run
		}
		lines = append(lines[:idx], append([]string{comment}, lines[idx:]...)...)
	}

	result := strings.Join(lines, "\n")

	for _, name := range methodNames {
		if existing[name] {
			continue
		}

		stub, err := stubFor(name)
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
