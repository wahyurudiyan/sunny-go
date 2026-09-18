// Package project scaffolds a new sgo project: the hexagonal directory
// tree described in ARCHITECTURE.md §3, conditioned on the HTTP
// framework, persistence mode, and datastores selected, plus the
// sgo.yaml manifest (ARCHITECTURE.md §7).
package project

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/bootstrap"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/httpgen"
	"github.com/wahyurudiyan/sunny-go/internal/config"
	sgotemplate "github.com/wahyurudiyan/sunny-go/internal/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Options describes the project to scaffold. It mirrors the selections
// persisted to sgo.yaml.
type Options struct {
	Name          string
	Module        string
	HTTPFramework config.HTTPFramework
	Persistence   config.Persistence
	Cache         []config.CacheEngine
	Search        []config.SearchEngine
}

// ValidateName checks that name is safe to use as both a directory name
// and a Go package/binary name.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	if strings.ContainsAny(name, " /\\:*?\"<>|") {
		return fmt.Errorf("project name contains invalid characters")
	}

	return nil
}

// Scaffold creates the project directory tree under destDir and writes
// sgo.yaml. destDir must not already exist.
func Scaffold(destDir string, opts Options) error {
	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("%s already exists", destDir)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	if err := createDirectories(destDir, opts); err != nil {
		return err
	}

	if err := renderFiles(destDir, opts); err != nil {
		return err
	}

	if err := writeDockerCompose(destDir, opts); err != nil {
		return err
	}

	httpDir := filepath.Join(destDir, "internal", "adapter", "in", "http", string(opts.HTTPFramework))
	if err := httpgen.GenerateServer(opts.HTTPFramework, httpDir); err != nil {
		return fmt.Errorf("failed to generate HTTP server boilerplate: %w", err)
	}

	bootstrapDir := filepath.Join(destDir, "internal", "bootstrap")
	if err := bootstrap.Generate(opts.Module, opts.HTTPFramework, nil, bootstrapDir); err != nil {
		return fmt.Errorf("failed to generate bootstrap: %w", err)
	}

	cfg := &config.Config{
		Module:        opts.Module,
		HTTPFramework: opts.HTTPFramework,
		Persistence:   opts.Persistence,
		Cache:         opts.Cache,
		Search:        opts.Search,
	}
	if err := cfg.Save(destDir); err != nil {
		return fmt.Errorf("failed to write %s: %w", config.FileName, err)
	}

	return gengo.TidyModule(destDir)
}

func createDirectories(destDir string, opts Options) error {
	dirs := []string{
		filepath.Join("contract", "pb"),
		filepath.Join("contract", "gen"),
		filepath.Join("internal", "core", "domain"),
		filepath.Join("internal", "core", "port", "in"),
		filepath.Join("internal", "core", "port", "out"),
		filepath.Join("internal", "core", "service"),
		filepath.Join("internal", "adapter", "in", "http", string(opts.HTTPFramework)),
		filepath.Join("internal", "adapter", "in", "grpc"),
		filepath.Join("internal", "adapter", "out"),
		filepath.Join("internal", "bootstrap"),
		filepath.Join("cmd", opts.Name),
		"docker",
	}

	for _, engine := range opts.Persistence.Engines {
		dirs = append(dirs, filepath.Join("internal", "adapter", "out", "persistence", string(engine)))
	}
	for _, engine := range opts.Cache {
		dirs = append(dirs, filepath.Join("internal", "adapter", "out", "cache", string(engine)))
	}
	for _, engine := range opts.Search {
		dirs = append(dirs, filepath.Join("internal", "adapter", "out", "search", string(engine)))
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(destDir, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func renderFiles(destDir string, opts Options) error {
	engine := sgotemplate.New(templatesFS)
	data := struct {
		Name   string
		Module string
	}{
		Name:   opts.Name,
		Module: opts.Module,
	}

	files := map[string]string{
		"templates/go.mod.tmpl":     filepath.Join(destDir, "go.mod"),
		"templates/Makefile.tmpl":   filepath.Join(destDir, "Makefile"),
		"templates/Dockerfile.tmpl": filepath.Join(destDir, "docker", "Dockerfile"),
		"templates/main.go.tmpl":    filepath.Join(destDir, "cmd", opts.Name, "main.go"),
	}

	for templatePath, dest := range files {
		if err := engine.Render(templatePath, data, dest); err != nil {
			return err
		}
	}

	return nil
}
