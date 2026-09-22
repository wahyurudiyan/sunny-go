// Package core generates the hexagonal core (ARCHITECTURE.md §3, §5,
// §6) for one entity from a compiled proto file: domain structs, the
// usecase and repository ports, the service skeleton (with the
// safe-regeneration guarantee that hand-written method bodies are never
// touched or removed), and the wire↔domain mapper.
package core

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// renderTemplate renders templatePath with data, unformatted. Used
// instead of internal/template.Engine here because a couple of call
// sites (safe-regen's appended method stub) need the raw rendered text
// in memory — not gofmt'd on its own, not written straight to a file —
// so it can be concatenated with other source before a single gofmt
// pass over the whole result.
func renderTemplate(templatePath string, data any) ([]byte, error) {
	content, err := templatesFS.ReadFile(templatePath)
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

	return buf.Bytes(), nil
}

// renderGo renders templatePath with data and gofmt's the result. Only
// valid for templates that render a complete Go source file on their
// own (everything except the method-stub fragment).
func renderGo(templatePath string, data any) ([]byte, error) {
	rendered, err := renderTemplate(templatePath, data)
	if err != nil {
		return nil, err
	}

	formatted, err := format.Source(rendered)
	if err != nil {
		return nil, fmt.Errorf("failed to gofmt output of %s: %w", templatePath, err)
	}

	return formatted, nil
}

// writeGoFile renders templatePath and writes it to destPath, creating
// any missing parent directories. It always overwrites destPath — only
// call it for generated (_gen.go) files.
func writeGoFile(templatePath string, data any, destPath string) error {
	formatted, err := renderGo(templatePath, data)
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

// Paths centralizes the Go import paths derived from a project's module
// path and an entity name, so every generator in this package agrees on
// where things live.
type Paths struct {
	Module string // sgo.yaml's module, e.g. "demo"
	Entity string // lowercase entity/service name, e.g. "user"
}

func (p Paths) DomainImportPath() string {
	return path.Join(p.Module, "internal/core/domain", p.Entity)
}

func (p Paths) PortInImportPath() string {
	return path.Join(p.Module, "internal/core/port/in")
}

func (p Paths) PortOutImportPath() string {
	return path.Join(p.Module, "internal/core/port/out")
}

func (p Paths) WireImportPath() string {
	return path.Join(p.Module, "contract/gen", p.Entity)
}

func (p Paths) MapperImportPath() string {
	return path.Join(p.Module, "internal/adapter/mapper")
}

// AggregateDomainImportPath is DomainImportPath's Phase 12 (ARCHITECTURE.md
// §17) successor: internal/domain/<entity>, not internal/core/domain/
// <entity>. Named distinctly from DomainImportPath while both exist
// side by side; GenerateCode switches over once the
// application/infrastructure generators exist too (§17's project-wide
// rewiring).
func (p Paths) AggregateDomainImportPath() string {
	return path.Join(p.Module, "internal/domain", p.Entity)
}

// EventImportPath is the shared internal/domain/event package
// (ARCHITECTURE.md §17) every entity's domain events implement
// DomainEvent through, so a single EventPublisher can accept events
// from any of them.
func (p Paths) EventImportPath() string {
	return path.Join(p.Module, "internal/domain/event")
}

// ApplicationImportPath is internal/application/<entity> — the CQRS
// command/query DTOs and application service (ARCHITECTURE.md §17).
func (p Paths) ApplicationImportPath() string {
	return path.Join(p.Module, "internal/application", p.Entity)
}

// ApplicationPortsImportPath is the shared internal/application/ports
// package — EventPublisher and any other cross-entity application-layer
// port.
func (p Paths) ApplicationPortsImportPath() string {
	return path.Join(p.Module, "internal/application/ports")
}

// entityTitle returns the exported (PascalCase-first-letter) form of an
// entity name, e.g. "user" -> "User". Proto message/service names in our
// own scaffold already follow this convention; this only matters when
// building strings that aren't themselves proto identifiers.
func entityTitle(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// primaryService returns the file's service, which sgo's generated code
// assumes is exactly one per proto file (matching the starter template
// from `sgo generate proto`).
func primaryService(f *sgoproto.File) (*sgoproto.Service, bool) {
	if len(f.Services) == 0 {
		return nil, false
	}
	return &f.Services[0], true
}
