// Package template renders text/template files from an embedded
// filesystem to disk. It replaces the old approach of storing generator
// templates as Go string constants (see ARCHITECTURE.md §9).
package template

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// Engine renders named templates from an embedded filesystem to disk.
type Engine struct {
	fs embed.FS
}

// New returns an Engine that reads templates from fs.
func New(fs embed.FS) *Engine {
	return &Engine{fs: fs}
}

// Render parses the template at templatePath (a path inside the embedded
// filesystem) and writes its output to destPath, creating any missing
// parent directories.
func (e *Engine) Render(templatePath string, data any, destPath string) error {
	content, err := e.fs.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", destPath, err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to render %s: %w", templatePath, err)
	}

	return nil
}
