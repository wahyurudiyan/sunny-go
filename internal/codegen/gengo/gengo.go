// Package gengo is a small shared helper for rendering an embedded
// text/template into gofmt'd Go source. Used by the generators under
// internal/codegen (httpgen, grpcgen, memgen, bootstrap) that each embed
// their own templates/*.tmpl but share the same render-then-write logic.
package gengo

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

// Render parses the template at templatePath (inside src) with data and
// gofmt's the result. Only valid for templates that produce a complete
// Go source file on their own.
func Render(src fs.FS, templatePath string, data any) ([]byte, error) {
	content, err := fs.ReadFile(src, templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render %s: %w", templatePath, err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to gofmt output of %s: %w", templatePath, err)
	}

	return formatted, nil
}

// Write renders templatePath and writes it to destPath, creating any
// missing parent directories. Always overwrites destPath — only call it
// for generated (_gen.go) files.
func Write(src fs.FS, templatePath string, data any, destPath string) error {
	formatted, err := Render(src, templatePath, data)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
	}

	if err := os.WriteFile(destPath, formatted, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", destPath, err)
	}

	return nil
}

// TidyModule runs `go mod tidy` in dir, so dependencies newly imported
// by generated code are picked up automatically. Used by both `sgo
// init` (gin/grpc imports from the HTTP/bootstrap boilerplate) and
// `sgo generate code` (protobuf/grpc imports from contract/gen and the
// adapters).
func TidyModule(dir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %w\n%s", err, out)
	}

	return nil
}
